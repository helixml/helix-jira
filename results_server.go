package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
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
	HelixYaml            string `json:"helix_yaml"`
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Helix Test Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; display: flex; flex-direction: column; height: 100vh; }
        .content { flex: 1; overflow-y: auto; padding-bottom: 10px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr.pass { background-color: #e6ffe6; }
        tr.fail { background-color: #ffe6e6; }
        h1 { color: #333; }
        #iframe-container { display: none; position: fixed; bottom: 0; left: 0; width: 100%; height: 70%; border: none; }
        #iframe-container iframe { width: 100%; height: calc(100% - 10px); border: none; }
        #close-iframe { position: absolute; top: 10px; right: 10px; cursor: pointer; }
        #resize-handle { width: 100%; height: 10px; background: #f0f0f0; cursor: ns-resize; border-top: 1px solid #ccc; }
        #view-helix-yaml { margin-bottom: 10px; }
    </style>
</head>
<body>
    <div class="content">
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
        <button id="view-helix-yaml" onclick="viewHelixYaml()">View helix.yaml</button>
        <table>
            <tr>
                <th>Test Name</th>
                <th>Result</th>
                <th>Session ID</th>
                <th>Model</th>
                <th>Inference Time</th>
                <th>Evaluation Time</th>
                <th>Session Link</th>
                <th>Debug Link</th>
            </tr>
            {{range .Tests}}
            <tr class="{{if eq .Result "PASS"}}pass{{else}}fail{{end}}">
                <td>{{.TestName}}</td>
                <td>{{.Result}}</td>
                <td>{{.SessionID}}</td>
                <td>{{.Model}}</td>
                <td>{{.InferenceTime}}</td>
                <td>{{.EvaluationTime}}</td>
                <td><a href="#" onclick="openDashboard('https://app.tryhelix.ai/session/{{.SessionID}}'); return false;">Session</a></td>
                <td><a href="#" onclick="openDashboard('https://app.tryhelix.ai/dashboard?tab=llm_calls&filter_sessions={{.SessionID}}'); return false;">Debug</a></td>
            </tr>
            {{end}}
        </table>
    </div>
    <div id="iframe-container">
        <div id="resize-handle"></div>
        <div id="close-iframe" onclick="closeDashboard()">Close</div>
        <iframe id="dashboard-iframe" src=""></iframe>
    </div>
    <script>
        function openDashboard(url) {
            document.getElementById('dashboard-iframe').src = url;
            document.getElementById('iframe-container').style.display = 'block';
        }
        function closeDashboard() {
            document.getElementById('iframe-container').style.display = 'none';
            document.getElementById('dashboard-iframe').src = '';
        }

        // Resizing functionality
        const resizeHandle = document.getElementById('resize-handle');
        const iframeContainer = document.getElementById('iframe-container');
        let isResizing = false;

        resizeHandle.addEventListener('mousedown', function(e) {
            isResizing = true;
            document.addEventListener('mousemove', resize);
            document.addEventListener('mouseup', stopResize);
        });

        function resize(e) {
            if (!isResizing) return;
            const newHeight = window.innerHeight - e.clientY;
            iframeContainer.style.height = newHeight + 'px';
        }

        function stopResize() {
            isResizing = false;
            document.removeEventListener('mousemove', resize);
        }

        function viewHelixYaml() {
            const helixYaml = {{.HelixYaml}};
            const blob = new Blob([helixYaml], { type: 'text/yaml' });
            const url = URL.createObjectURL(blob);
            openDashboard(url);
        }
    </script>
</body>
</html>
`

func RunResultsServer() {
	fmt.Println("Starting results server...")
	startServer()
}

func startServer() {
	http.HandleFunc("/", handleResults)
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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

	if len(data.Tests) > 0 {
		data.HelixYaml = data.Tests[0].HelixYaml
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
