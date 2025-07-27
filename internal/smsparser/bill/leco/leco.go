package leco

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LecoParser struct{}

var (
	dateLayout      = "02-Jan-06"
	amountRegex     = regexp.MustCompile(`[-+]?[\d,]+(?:\.\d+)?`)
	readingRegex    = regexp.MustCompile(`(\d+)\s*-\s*(\d+)\s*=\s*(\d+)`)
	netUnitsRegex   = regexp.MustCompile(`([-\d]+)\s*\((\w+)\)`)
	errInvalidValue = errors.New("invalid value")
)

func (*LecoParser) Parse(sms string) (*ElectricityBill, error) {
	bill := &ElectricityBill{}
	lines := strings.Split(sms, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Normalize line by removing prefix markers
		if strings.HasPrefix(line, ">") {
			line = strings.TrimSpace(line[1:])
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		var err error
		switch key {
		case "Read On":
			bill.ReadOn, err = parseDate(value)
		case "Imp":
			err = parseReading(value, &bill.ImportReadingPrevious, &bill.ImportReadingCurrent, &bill.ImportUnits)
		case "Exp":
			err = parseReading(value, &bill.ExportReadingPrevious, &bill.ExportReadingCurrent, &bill.ExportUnits)
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

		if err != nil {
			return nil, fmt.Errorf("error parsing '%s': %w", key, err)
		}
	}
	return bill, nil
}

func parseDate(s string) (time.Time, error) {
	// Normalize month abbreviation to title case
	if parts := strings.Split(s, "-"); len(parts) == 3 {
		s = fmt.Sprintf("%s-%s-%s", parts[0], strings.Title(strings.ToLower(parts[1])), parts[2])
	}
	return time.Parse(dateLayout, s)
}

func parseReading(s string, prev, curr, units *int) error {
	matches := readingRegex.FindStringSubmatch(s)
	if len(matches) < 4 {
		return fmt.Errorf("%w: reading format", errInvalidValue)
	}

	var err error
	*prev, err = strconv.Atoi(matches[1])
	if err != nil {
		return err
	}

	*curr, err = strconv.Atoi(matches[2])
	if err != nil {
		return err
	}

	*units, err = strconv.Atoi(matches[3])
	return err
}

func parseNetUnits(s string, units *int, unitType *string) error {
	matches := netUnitsRegex.FindStringSubmatch(s)
	if len(matches) < 3 {
		return fmt.Errorf("%w: net units format", errInvalidValue)
	}

	var err error
	*units, err = strconv.Atoi(matches[1])
	if err != nil {
		return err
	}

	*unitType = matches[2]
	return nil
}

func parseAmount(s string) (float64, error) {
	// Extract first numeric value found
	if match := amountRegex.FindString(s); match != "" {
		clean := strings.ReplaceAll(match, ",", "")
		return strconv.ParseFloat(clean, 64)
	}
	return 0, fmt.Errorf("%w: amount format", errInvalidValue)
}

func parseBalanceWithDate(s string) (float64, time.Time, error) {
	parts := strings.Split(s, " on ")
	if len(parts) < 1 {
		return 0, time.Time{}, fmt.Errorf("%w: balance format", errInvalidValue)
	}

	amount, err := parseAmount(parts[0])
	if err != nil {
		return 0, time.Time{}, err
	}

	var date time.Time
	if len(parts) > 1 {
		date, err = parseDate(parts[1])
		if err != nil {
			return amount, time.Time{}, err
		}
	}

	return amount, date, nil
}

func parsePayment(s string) (float64, time.Time, error) {
	parts := strings.Split(s, " on ")
	if len(parts) < 1 {
		return 0, time.Time{}, fmt.Errorf("%w: payment format", errInvalidValue)
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
