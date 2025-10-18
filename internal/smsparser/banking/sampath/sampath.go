package sampath

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"auto-finance/internal/models/finance"
	"auto-finance/internal/smsparser"
)

var (
	// cardTxnRegex matches card-based transactions. It intentionally does not
	// assert a word boundary before "Cr" because some SMS strings may
	// concatenate the date with "Cr" (e.g. "17-OCTCr"), leaving no
	// whitespace separator. The pattern looks for "Cr Crd no..**<digits>" and
	// then captures the transaction status (ignored), the currency, the
	// amount and the merchant name. Matching is case-insensitive. The final
	// merchant capture stops at either "Avl Bal", "Enq" or end-of-string.
	cardTxnRegex = regexp.MustCompile(`(?i)Cr\s+Crd\s+no\.{2}\*\*(\d{3,5})\s+\w+\s+Pmt\s+([A-Z]{3})\s+([\d,.]+|\.00)\s+at\s+(.+?)\s+(?:Avl\s+Bal|Enq|$)`)
	// accountCreditedRegex matches account credit notifications where funds
	// are credited to an account (AC **<digits>). It captures the currency,
	// amount, account digits and merchant/description. Matching is
	// case-insensitive. The merchant capture stops at a space followed by a
	// dash or the end of the string. A dash often precedes additional bank
	// messaging.
	accountCreditedRegex = regexp.MustCompile(`(?i)([A-Z]{3})\s+([\d,.]+)\s+credited\s+to\s+AC\s+\*\*(\d{3,5})\s+for\s+(.+?)(?:\s+-|$)`)
)

type parser struct{}

func New() smsparser.SMSParser[*finance.SampathModel] {
	return &parser{}
}

func (p *parser) GetName() string {
	return "Sampath Bank Parser"
}

func (p *parser) Parse(sms string) (*finance.SampathModel, error) {
	// Normalize whitespace to make regex matching more predictable and trim
	// leading/trailing spaces. Newlines and tabs are collapsed into single
	// spaces. We do not change the case since our regexes are case-insensitive.
	cleaned := strings.TrimSpace(sms)
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	// Attempt to match a card-based transaction first. This pattern captures
	// the card digits, currency, amount and merchant name.
	if matches := cardTxnRegex.FindStringSubmatch(cleaned); len(matches) == 5 {
		cardDigits := matches[1]
		currency := strings.ToUpper(matches[2])
		amtStr := matches[3]
		merchant := strings.TrimSpace(matches[4])

		// Remove any commas from the amount string to allow parsing to float.
		amtStr = strings.ReplaceAll(amtStr, ",", "")
		// Some USD notifications may show as ".00" if the amount is zero (e.g. USD .00)
		if amtStr == ".00" || amtStr == ".0" || amtStr == "." {
			amtStr = "0.00"
		}
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return nil, err
		}
		// Merchant strings sometimes include tildes (~) which represent
		// separators in the SMS. Replace with spaces for readability.
		merchant = strings.ReplaceAll(merchant, "~", " ")
		return &finance.SampathModel{
			TransactionType: finance.TransactionTypeCard,
			Identifier:      cardDigits,
			Amount:          amount,
			Currency:        currency,
			Merchant:        merchant,
		}, nil
	}

	// If no card match, attempt to match an account credit pattern.
	if matches := accountCreditedRegex.FindStringSubmatch(cleaned); len(matches) == 5 {
		currency := strings.ToUpper(matches[1])
		amtStr := matches[2]
		accountDigits := matches[3]
		merchant := strings.TrimSpace(matches[4])

		amtStr = strings.ReplaceAll(amtStr, ",", "")
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return nil, err
		}
		merchant = strings.ReplaceAll(merchant, "~", " ")
		return &finance.SampathModel{
			TransactionType: finance.TransactionTypeOnline,
			Amount:          amount,
			Currency:        currency,
			Identifier:      accountDigits,
			Merchant:        merchant,
		}, nil
	}

	// If neither pattern matches, return an error.
	return nil, errors.New("unrecognized SMS format")
}
