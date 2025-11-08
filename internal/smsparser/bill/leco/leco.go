package leco

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	models "auto-finance/internal/models/ebill"
	"auto-finance/internal/smsparser"
)

type parser struct{}

var (
	dateLayout     = "02-Jan-06"
	isoDateLayouts = []string{"2006-01-02", "2006/01/02"}
	amountRegex    = regexp.MustCompile(`[-+]?[\d,]+(?:\.\d+)?`)
	readingRegex   = regexp.MustCompile(`(\d+)\s*-\s*(\d+)\s*=\s*(\d+)`)
	netUnitsRegex  = regexp.MustCompile(`([-\d]+)(?:\s*\(([^)]+)\))?`)
	accountRegex   = regexp.MustCompile(`(\d+)\s*(?:\(([^)]+)\)|([A-Za-z][\w\s/-]*))?`)

	// Custom errors
	ErrInvalidValue   = errors.New("invalid value")
	ErrInvalidDate    = errors.New("invalid date format")
	ErrInvalidReading = errors.New("invalid reading format")
	ErrInvalidAmount  = errors.New("invalid amount format")
)

func New() smsparser.SMSParser[*models.ElectricityBill] {
	return &parser{}
}

func (*parser) Parse(sms string) (*models.ElectricityBill, error) {
	bill := &models.ElectricityBill{}
	lines := strings.Split(sms, "\n")

	var (
		parseErr        error
		pendingAcctName bool
		monthlyBillSet  bool
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
		normalizedKey := strings.ToLower(key)

		var err error
		switch normalizedKey {
		case "a/n":
			err = parseAccount(value, bill)
			pendingAcctName = (err == nil)
		case "read on", "reading date":
			bill.ReadOn, err = parseDate(value)
		case "imp", "import", "import reading":
			err = parseReading(value, &bill.ImportPrevious, &bill.ImportCurrent, &bill.ImportUnits)
		case "exp", "export", "export reading":
			err = parseReading(value, &bill.ExportPrevious, &bill.ExportCurrent, &bill.ExportUnits)
		case "net units":
			err = parseNetUnits(value, &bill.NetUnits, &bill.NetUnitsType)
		case "monthly bill":
			bill.MonthlyBill, err = parseAmount(value)
			if err == nil {
				monthlyBillSet = true
			}
		case "fixed charge":
			if !monthlyBillSet {
				bill.MonthlyBill, err = parseAmount(value)
				if err == nil {
					monthlyBillSet = true
				}
			}
		case "other charges":
			bill.OtherCharges, err = parseAmount(value)
		case "sscl", "ssc levy":
			bill.SSCL, err = parseAmount(value)
		case "opening balance":
			bill.OpeningBalance, bill.OpeningBalanceDate, err = parseBalanceWithDate(value)
		case "current outstanding amount":
			bill.OpeningBalance, err = parseAmount(value)
			bill.OpeningBalanceDate = time.Time{}
		case "total payable", "total due":
			bill.TotalPayable, err = parseAmount(value)
		case "last payment":
			bill.LastPaymentAmount, bill.LastPaymentDate, err = parsePayment(value)
		case "last amount paid for generation":
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

	if err := validateBill(bill); err != nil {
		return nil, fmt.Errorf("failed to validate bill: %w", err)
	}

	return bill, parseErr
}

func (p *parser) GetName() string {
	return "Leco SMS Parser"
}

func parseAccount(s string, bill *models.ElectricityBill) error {
	matches := accountRegex.FindStringSubmatch(s)
	if len(matches) < 2 {
		return fmt.Errorf("%w: account format", ErrInvalidValue)
	}

	bill.AccountNumber = matches[1]
	var accountType string
	if len(matches) > 2 && strings.TrimSpace(matches[2]) != "" {
		accountType = matches[2]
	} else if len(matches) > 3 {
		accountType = matches[3]
	}
	bill.AccountType = strings.TrimSpace(accountType)
	return nil
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range isoDateLayouts {
		if len(s) >= len(layout) && (strings.Contains(s, "-") || strings.Contains(s, "/")) {
			if t, err := time.Parse(layout, s); err == nil {
				return t, nil
			}
		}
	}

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
	first, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("%w: previous reading", ErrInvalidReading)
	}

	second, err := strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("%w: current reading", ErrInvalidReading)
	}

	parsedUnits, err := strconv.Atoi(matches[3])
	if err != nil {
		return fmt.Errorf("%w: units calculation", ErrInvalidReading)
	}

	if first <= second {
		*prev = first
		*curr = second
	} else {
		*prev = second
		*curr = first
	}

	*units = parsedUnits
	return nil
}

func parseNetUnits(s string, units *int, unitType *string) error {
	matches := netUnitsRegex.FindStringSubmatch(s)
	if len(matches) < 2 {
		return fmt.Errorf("%w: net units format", ErrInvalidValue)
	}

	var err error
	*units, err = strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("%w: net units value", ErrInvalidValue)
	}

	if len(matches) > 2 {
		*unitType = strings.TrimSpace(matches[2])
	} else {
		*unitType = ""
	}
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

func validateBill(bill *models.ElectricityBill) error {
	if bill.AccountNumber == "" {
		return errors.New("account number is required")
	}
	if bill.ReadOn.IsZero() {
		return errors.New("read date is required")
	}
	if bill.ImportUnits < 0 {
		return errors.New("import units must be non-negative")
	}
	if bill.ExportUnits < 0 {
		return errors.New("export units must be non-negative")
	}
	if bill.MonthlyBill < 0 {
		return errors.New("monthly bill must be non-negative")
	}
	return nil
}
