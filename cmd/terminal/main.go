package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// / initSheetsService initializes the Google Sheets service client.
// It uses a service account key file for authentication.
func initSheetsService(ctx context.Context, serviceAccountFile string) (*sheets.Service, error) {
	// Authenticate using the service account key file.
	// The option.WithServiceAccountFile function reads the JSON key file
	// and sets up the credentials.
	srv, err := sheets.NewService(ctx, option.WithScopes(sheets.SpreadsheetsScope), option.WithServiceAccountFile(serviceAccountFile))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}
	return srv, nil
}

// writeToSheet appends data to a specified range in a Google Sheet.
func writeToSheet(srv *sheets.Service, spreadsheetID, sheetRange string, values [][]interface{}) error {
	// Create a ValueRange object with the data to be written.
	// Each inner slice represents a row, and each element in the inner slice is a cell value.
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	// Call the Append method to add data to the sheet.
	// The "USER_ENTERED" value input option means that the data will be parsed
	// as if it were entered by a user (e.g., numbers will be parsed as numbers, dates as dates).
	// The "INSERT_ROWS" insert data option means new rows will be inserted at the end of the sheet.
	_, err := srv.Spreadsheets.Values.Append(spreadsheetID, sheetRange, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		return fmt.Errorf("unable to append data to sheet: %v", err)
	}

	fmt.Printf("Data successfully written to spreadsheet ID: %s, range: %s\n", spreadsheetID, sheetRange)
	return nil
}

func main() {
	// Set up the context for the API calls.
	ctx := context.Background()

	// --- Configuration ---
	// Replace with the path to your service account key JSON file.
	// Make sure this file is secure and not committed to version control.
	serviceAccountFile := ".key"

	// Replace with the ID of your Google Spreadsheet.
	// You can find this in the URL of your spreadsheet:
	// https://docs.google.com/spreadsheets/d/YOUR_SPREADSHEET_ID/edit
	spreadsheetID := "1sM6wKz2pVlus-fZ8qbDCGmJBYqRxwE3XfhLv1kn1-J4"

	// Replace with the name of the sheet and the range where you want to write data.
	// For example, "Sheet1!A1" will start appending from cell A1 of Sheet1.
	sheetRange := "Sheet1!A1"
	// --- End Configuration ---

	// Initialize the Google Sheets service.
	srv, err := initSheetsService(ctx, serviceAccountFile)
	if err != nil {
		log.Fatalf("Error initializing Sheets service: %v", err)
	}

	// Data to be written to the Google Sheet.
	// Each inner slice is a row.
	dataToWrite := [][]interface{}{
		{"Name", "Email", "Date Joined"},
		{"Alice Smith", "alice@example.com", "2023-01-15"},
		{"Bob Johnson", "bob@example.com", "2023-02-20"},
		{"Charlie Brown", "charlie@example.com", "2023-03-10"},
	}

	// Write the data to the Google Sheet.
	err = writeToSheet(srv, spreadsheetID, sheetRange, dataToWrite)
	if err != nil {
		log.Fatalf("Error writing data to sheet: %v", err)
	}

	fmt.Println("Application finished successfully.")
}
