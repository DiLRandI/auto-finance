package leco

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"auto-finance/internal/smsparser"
)

func New() smsparser.SMSParser[*ElectricityBill] {
	return &parser{}
}

type parser struct{}

var (
	dateLayout    = "02-Jan-06"
	amountRegex   = regexp.MustCompile(`[-+]?[\d,]+(?:\.\d+)?`)
	readingRegex  = regexp.MustCompile(`(\d+)\s*-\s*(\d+)\s*=\s*(\d+)`)
	netUnitsRegex = regexp.MustCompile(`([-\d]+)\s*\((\w+)\)`)
	accountRegex  = regexp.MustCompile(`(\d+)\s*(?:\(([^)]+)\))?`)

	// Custom errors
	ErrInvalidValue   = errors.New("invalid value")
	ErrInvalidDate    = errors.New("invalid date format")
	ErrInvalidReading = errors.New("invalid reading format")
	ErrInvalidAmount  = errors.New("invalid amount format")
)

func (*parser) Parse(sms string) (*ElectricityBill, error) {
	bill := &ElectricityBill{}
	lines := strings.Split(sms, "\n")

	var (
		parseErr        error
		pendingAcctName bool
	)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle both prefixed and non-prefixed lines
		line = strings.TrimPrefix(line, ">")
		line = strings.TrimSpace(line)

		// Handle pending account name from previous A/N line
		if pendingAcctName {
			if !strings.Contains(line, ":") && line != "" {
				bill.AccountName = line
				pendingAcctName = false
				continue
			} else {
				pendingAcctName = false
			}
		}

		// Skip non-key:value lines except account name
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		var err error
		switch key {
		case "A/N":
			err = parseAccount(value, bill)
			pendingAcctName = (err == nil)
		case "Read On":
			bill.ReadOn, err = parseDate(value)
		case "Imp":
			err = parseReading(value, &bill.ImportPrevious, &bill.ImportCurrent, &bill.ImportUnits)
		case "Exp":
			err = parseReading(value, &bill.ExportPrevious, &bill.ExportCurrent, &bill.ExportUnits)
		case "Net Units":
			err = parseNetUnits(value, &bill.NetUnits, &bill.NetUnitsType)
		case "Monthly Bill":
			bill.MonthlyBill, err = parseAmount(value)
		case "Other Charges":
			bill.OtherCharges, err = parseAmount(value)
		case "SSCL":
			bill.SSCL, err = parseAmount(value)
		case "Opening Balance":
			bill.OpeningBalance, bill.OpeningBalanceDate, err = parseBalanceWithDate(value)
		case "Total Payable":
			bill.TotalPayable, err = parseAmount(value)
		case "Last Payment":
			bill.LastPaymentAmount, bill.LastPaymentDate, err = parsePayment(value)
		case "Last Amount Paid for Generation":
			bill.LastGenPayment, err = parseAmount(value)
		}

		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("%s: %w", key, err)
		}
	}

	if pendingAcctName {
		if parseErr == nil {
			parseErr = errors.New("account name missing after A/N")
		}
	}

	return bill, parseErr
}

func parseAccount(s string, bill *ElectricityBill) error {
	matches := accountRegex.FindStringSubmatch(s)
	if len(matches) < 2 {
		return fmt.Errorf("%w: account format", ErrInvalidValue)
	}

	bill.AccountNumber = matches[1]
	if len(matches) > 2 {
		bill.AccountType = matches[2]
	}
	return nil
}

func parseDate(s string) (time.Time, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("%w: expected DD-MMM-YY format", ErrInvalidDate)
	}

	// Normalize month: "JUL" -> "Jul"
	month := strings.ToUpper(parts[1])
	if len(month) >= 3 {
		month = strings.ToUpper(month[0:1]) + strings.ToLower(month[1:3])
	}
	dateStr := fmt.Sprintf("%s-%s-%s", parts[0], month, parts[2])

	t, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", ErrInvalidDate, err)
	}
	return t, nil
}

func parseReading(s string, prev, curr, units *int) error {
	matches := readingRegex.FindStringSubmatch(s)
	if len(matches) != 4 {
		return fmt.Errorf("%w: expected format '123-456=789'", ErrInvalidReading)
	}

	var err error
	*prev, err = strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("%w: previous reading", ErrInvalidReading)
	}

	*curr, err = strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("%w: current reading", ErrInvalidReading)
	}

	*units, err = strconv.Atoi(matches[3])
	if err != nil {
		return fmt.Errorf("%w: units calculation", ErrInvalidReading)
	}

	return nil
}

func parseNetUnits(s string, units *int, unitType *string) error {
	matches := netUnitsRegex.FindStringSubmatch(s)
	if len(matches) != 3 {
		return fmt.Errorf("%w: net units format", ErrInvalidValue)
	}

	var err error
	*units, err = strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("%w: net units value", ErrInvalidValue)
	}

	*unitType = matches[2]
	return nil
}

func parseAmount(s string) (float64, error) {
	match := amountRegex.FindString(s)
	if match == "" {
		return 0, fmt.Errorf("%w: no numeric value found", ErrInvalidAmount)
	}

	clean := strings.ReplaceAll(match, ",", "")
	val, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
	}

	return val, nil
}

func parseBalanceWithDate(s string) (float64, time.Time, error) {
	parts := strings.Split(s, " on ")
	if len(parts) < 1 {
		return 0, time.Time{}, fmt.Errorf("%w: balance format", ErrInvalidValue)
	}

	amount, err := parseAmount(parts[0])
	if err != nil {
		return 0, time.Time{}, err
	}

	var date time.Time
	if len(parts) > 1 {
		date, err = parseDate(parts[1])
	}

	return amount, date, err
}

func parsePayment(s string) (float64, time.Time, error) {
	parts := strings.Split(s, " on ")
	if len(parts) < 1 {
		return 0, time.Time{}, fmt.Errorf("%w: payment format", ErrInvalidValue)
	}

	amount, err := parseAmount(parts[0])
	if err != nil {
		return 0, time.Time{}, err
	}

	var date time.Time
	if len(parts) > 1 {
		date, err = parseDate(parts[1])
	}

	return amount, date, err
}
