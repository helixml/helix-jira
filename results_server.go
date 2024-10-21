package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

type ResultsData struct {
	Tests                []TestResult `json:"tests"`
	TotalExecutionTime   string       `json:"total_execution_time"`
	LatestResultsFile    string
	AvailableResultFiles []string
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Helix Test Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        h1 { color: #333; }
    </style>
</head>
<body>
    <h1>Helix Test Results</h1>
    <p>Total Execution Time: {{.TotalExecutionTime}}</p>
    <p>Current Results File: {{.LatestResultsFile}}</p>
    <form action="/" method="get">
        <select name="file" onchange="this.form.submit()">
            {{range .AvailableResultFiles}}
                <option value="{{.}}" {{if eq . $.LatestResultsFile}}selected{{end}}>{{.}}</option>
            {{end}}
        </select>
    </form>
    <table>
        <tr>
            <th>Test Name</th>
            <th>Result</th>
            <th>Session ID</th>
            <th>Inference Time</th>
            <th>Evaluation Time</th>
            <th>Link</th>
        </tr>
        {{range .Tests}}
        <tr>
            <td>{{.TestName}}</td>
            <td>{{.Result}}</td>
            <td>{{.SessionID}}</td>
            <td>{{.InferenceTime}}</td>
            <td>{{.EvaluationTime}}</td>
            <td><a href="https://app.tryhelix.ai/dashboard?tab=llm_calls&filter_sessions={{.SessionID}}" target="_blank">View</a></td>
        </tr>
        {{end}}
    </table>
</body>
</html>
`

func RunResultsServer() {
	http.HandleFunc("/", handleResults)
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func handleResults(w http.ResponseWriter, r *http.Request) {
	resultsFile := r.URL.Query().Get("file")
	if resultsFile == "" {
		resultsFile = getLatestResultsFile()
	}

	data, err := loadResultsData(resultsFile)
	if err != nil {
		http.Error(w, "Error loading results data", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("results").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func loadResultsData(filename string) (ResultsData, error) {
	var data ResultsData

	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return data, err
	}

	data.LatestResultsFile = filename
	data.AvailableResultFiles = getAvailableResultFiles()

	return data, nil
}

func getLatestResultsFile() string {
	files, err := filepath.Glob("results_*.json")
	if err != nil || len(files) == 0 {
		return ""
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files[0]
}

func getAvailableResultFiles() []string {
	files, err := filepath.Glob("results_*.json")
	if err != nil {
		return []string{}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files
}
