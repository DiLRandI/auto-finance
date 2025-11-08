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
	// cardAuthRegex matches authorization, reversal, and decline card payment notifications.
	// Captures: [1] last digits, [2] status token, [3] currency, [4] amount, [5] merchant.
	cardAuthRegex = regexp.MustCompile(`(?i)\w+\s+Crd\s+no\.{2}\*\*(\d{3,5})\s+(\w+)\s+Pmt\s+([A-Z]{3})\s+([\d,.]+|\.00)\s+at\s+(.+?)\s+(?:Avl\s+Bal|Enq|$)`)
	// cardCreditRegex matches statement credits (e.g. payment received) that use "Credited ... for ...".
	// Captures: [1] last digits, [2] currency, [3] amount, [4] description.
	cardCreditRegex = regexp.MustCompile(`(?i)\w+\s+Crd\s+no\.{2}\*\*(\d{3,5})\s+Credited\s+([A-Z]{3})\s+([\d,.]+|\.00)\s+for\s+(.+?)\s+(?:Avl\s+Bal|Enq|$)`)
	// accountTxnRegex matches account-based transactions (credits, debits, ATM withdrawals).
	// Captures: [1] currency, [2] amount, [3] txn type token, [4] account digits,
	// [5] channel token ("for" vs "via ATM at"), [6] description/merchant.
	accountTxnRegex = regexp.MustCompile(`(?i)^([A-Z]{3})\s+([\d,.]+)\s+(credited\s+to|debited\s+from)\s+AC\s+\*\*(\d{3,5})\s+(via\s+ATM\s+at|for)\s+(.+?)(?:\s+(?:For\s+Inq|For\s+Enq|Enq)\b.*)?$`)
	// avlBalRegex captures "Avl Bal <currency> <amount>" fragments present in credit card SMS alerts.
	avlBalRegex = regexp.MustCompile(`(?i)Avl\s+Bal\s+([A-Z]{3})\s+([\d,.]+|\.00)`)
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

	if matches := cardAuthRegex.FindStringSubmatch(cleaned); len(matches) == 6 {
		cardDigits := matches[1]
		statusCode := matches[2]
		currency := strings.ToUpper(matches[3])
		amtStr := normalizeAmount(matches[4])
		merchant := strings.TrimSpace(matches[5])

		// Normalize amount string and convert to float.
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
		model := &finance.SampathModel{
			TransactionType: finance.TransactionTypeCard,
			Identifier:      cardDigits,
			Amount:          amount,
			Currency:        currency,
			Merchant:        merchant,
			Status:          status,
			SmsDateTime:     time.Now().Format(time.DateTime),
		}
		applyAvailableBalance(model, cleaned)
		return model, nil
	}

	if matches := cardCreditRegex.FindStringSubmatch(cleaned); len(matches) == 5 {
		cardDigits := matches[1]
		currency := strings.ToUpper(matches[2])
		amtStr := normalizeAmount(matches[3])
		amount, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return nil, err
		}
		description := strings.ReplaceAll(strings.TrimSpace(matches[4]), "~", " ")

		model := &finance.SampathModel{
			TransactionType: finance.TransactionTypeCard,
			Identifier:      cardDigits,
			Amount:          amount,
			Currency:        currency,
			Merchant:        description,
			Status:          "credit",
			SmsDateTime:     time.Now().Format(time.DateTime),
		}
		applyAvailableBalance(model, cleaned)
		return model, nil
	}

	// If no card match, attempt to match an account transaction (credit or debit).
	// matches: [0]=full, [1]=currency, [2]=amount, [3]=txnType, [4]=accountDigits, [5]=channel, [6]=description
	if matches := accountTxnRegex.FindStringSubmatch(cleaned); len(matches) == 7 {
		currency := strings.ToUpper(matches[1])
		amtStr := normalizeAmount(matches[2])
		txnType := strings.ToLower(strings.TrimSpace(matches[3]))
		accountDigits := matches[4]
		channel := strings.ToLower(strings.TrimSpace(matches[5]))
		description := strings.TrimSpace(matches[6])

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

		transactionType := finance.TransactionTypeOnline
		if strings.HasPrefix(channel, "via atm") {
			transactionType = finance.TransactionTypeATM
		}

		return &finance.SampathModel{
			TransactionType: transactionType,
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

func normalizeAmount(raw string) string {
	clean := strings.ReplaceAll(raw, ",", "")
	switch clean {
	case ".00", ".0", ".":
		return "0.00"
	}
	return clean
}

func applyAvailableBalance(model *finance.SampathModel, sms string) {
	if currency, amount, ok := parseAvailableBalance(sms); ok {
		model.AvailableBalanceCurrency = currency
		model.AvailableBalance = amount
	}
}

func parseAvailableBalance(sms string) (string, float64, bool) {
	matches := avlBalRegex.FindStringSubmatch(sms)
	if len(matches) != 3 {
		return "", 0, false
	}

	currency := strings.ToUpper(matches[1])
	amount, err := strconv.ParseFloat(normalizeAmount(matches[2]), 64)
	if err != nil {
		return "", 0, false
	}
	return currency, amount, true
}
