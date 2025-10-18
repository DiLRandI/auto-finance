package sampath

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"auto-finance/internal/models/finance"
	"auto-finance/internal/smsparser"
)

var (
	// cardTxnRegex matches card-based transactions. It does not assert any
	// specific prefix before "Crd"; the prefix may be "Cr", "Web" or
	// another alpha string indicating the channel (e.g. web purchases). The
	// pattern looks for "<prefix> Crd no..**<digits>" and captures the
	// transaction status, currency, amount and merchant name. The merchant
	// capture stops at either "Avl Bal", "Enq" or end-of-string.
	// Capturing groups:
	// [1] card last digits, [2] status code, [3] currency, [4] amount, [5] merchant.
	cardTxnRegex = regexp.MustCompile(`(?i)\w+\s+Crd\s+no\.{2}\*\*(\d{3,5})\s+(\w+)\s+Pmt\s+([A-Z]{3})\s+([\d,.]+|\.00)\s+at\s+(.+?)\s+(?:Avl\s+Bal|Enq|$)`)
	// accountTxnRegex matches account-based transactions (credits or debits).
	// It captures the currency, amount, transaction type (credited to or
	// debited from), the last digits of the account and the description or
	// merchant. The description stops at the first space followed by a dash
	// (often preceding additional bank messaging) or at end-of-string. The
	// pattern also accommodates "via ATM at" constructs used for cash
	// withdrawals.
	accountTxnRegex = regexp.MustCompile(`(?i)([A-Z]{3})\s+([\d,.]+)\s+(credited\s+to|debited\s+from)\s+AC\s+\*\*(\d{3,5})\s+(?:for|via\s+ATM\s+at)\s+(.+?)(?:\s+-|$)`)
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

	if matches := cardTxnRegex.FindStringSubmatch(cleaned); len(matches) == 6 {
		cardDigits := matches[1]
		statusCode := matches[2]
		currency := strings.ToUpper(matches[3])
		amtStr := matches[4]
		merchant := strings.TrimSpace(matches[5])

		// Normalize amount string and convert to float.
		amtStr = strings.ReplaceAll(amtStr, ",", "")
		if amtStr == ".00" || amtStr == ".0" || amtStr == "." {
			amtStr = "0.00"
		}
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return nil, err
		}

		status := ""
		lc := strings.ToLower(statusCode)
		switch {
		case strings.HasPrefix(lc, "dcl"):
			status = "decline"
		case strings.HasPrefix(lc, "rvs"):
			status = "reversed"
		case strings.HasPrefix(lc, "aut"):
			status = "authorized"
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
			Status:          status,
			SmsDateTime:     time.Now().Format(time.DateTime),
		}, nil
	}

	// If no card match, attempt to match an account transaction (credit or debit).
	// matches: [0]=full, [1]=currency, [2]=amount, [3]=txnType, [4]=accountDigits, [5]=description
	if matches := accountTxnRegex.FindStringSubmatch(cleaned); len(matches) == 6 {
		currency := strings.ToUpper(matches[1])
		amtStr := matches[2]
		txnType := strings.ToLower(strings.TrimSpace(matches[3]))
		accountDigits := matches[4]
		description := strings.TrimSpace(matches[5])

		amtStr = strings.ReplaceAll(amtStr, ",", "")
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return nil, err
		}
		description = strings.ReplaceAll(description, "~", " ")

		// Derive a status: credit or debit
		status := ""
		if strings.HasPrefix(txnType, "credited") {
			status = "credit"
		} else if strings.HasPrefix(txnType, "debited") {
			status = "debit"
		}

		return &finance.SampathModel{
			TransactionType: finance.TransactionTypeOnline,
			Identifier:      accountDigits,
			Amount:          amount,
			Currency:        currency,
			Merchant:        description,
			Status:          status,
			SmsDateTime:     time.Now().Format(time.DateTime),
		}, nil
	}

	// If neither pattern matches, return an error.
	return nil, errors.New("unrecognized SMS format")
}
