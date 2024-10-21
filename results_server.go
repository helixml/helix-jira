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
        body, html { 
            font-family: Arial, sans-serif; 
            margin: 0; 
            padding: 0; 
            height: 100%; 
            overflow: hidden; 
        }
        .main-container {
            display: flex;
            flex-direction: column;
            height: 100vh;
        }
        .header { 
            padding: 10px 20px; 
            background-color: #f8f8f8;
            border-bottom: 1px solid #ddd;
            display: flex;
            align-items: center;
            justify-content: space-between;
            flex-wrap: wrap;
        }
        .header h1 {
            margin: 0;
            font-size: 1.2em;
        }
        .header-info {
            display: flex;
            align-items: center;
            gap: 20px;
        }
        .header-info p {
            margin: 0;
            font-size: 0.9em;
        }
        .header-controls {
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .results-container { 
            flex: 1;
            overflow-y: auto;
            padding: 0 20px;
        }
        table { 
            border-collapse: collapse; 
            width: 100%; 
        }
        th, td { 
            border: 1px solid #ddd; 
            padding: 8px; 
            text-align: left; 
        }
        th { 
            background-color: #f2f2f2; 
            position: sticky;
            top: 0;
            z-index: 10;
        }
        tr.pass { background-color: #e6ffe6; }
        tr.fail { background-color: #ffe6e6; }
        #iframe-container { 
            display: none; 
            position: fixed; 
            bottom: 0; 
            left: 0; 
            width: 100%; 
            height: 70%; 
            border: none; 
        }
        #iframe-container iframe { 
            width: 100%; 
            height: calc(100% - 10px); 
            border: none; 
        }
        #close-iframe { 
            position: absolute; 
            top: 10px; 
            right: 10px; 
            cursor: pointer; 
        }
        #resize-handle { 
            width: 100%; 
            height: 10px; 
            background: #f0f0f0; 
            cursor: ns-resize; 
            border-top: 1px solid #ccc; 
        }
        #view-helix-yaml { 
            padding: 5px 10px;
            font-size: 0.9em;
        }
        .truncate { 
            max-width: 400px; 
            white-space: nowrap; 
            overflow: hidden; 
            text-overflow: ellipsis; 
            position: relative;
            cursor: pointer;
        }
        .tooltip {
            display: none;
            position: absolute;
            background-color: #f9f9f9;
            border: 1px solid #ddd;
            padding: 5px;
            z-index: 1000;
            max-width: 300px;
            word-wrap: break-word;
            box-shadow: 0 2px 5px rgba(0,0,0,0.2);
        }
    </style>
</head>
<body>
    <div class="main-container">
        <div class="header">
            <h1>Helix Test Results</h1>
            <div class="header-info">
                <p>Total Time: {{.TotalExecutionTime}}</p>
                <p>File: {{.LatestResultsFile}}</p>
            </div>
            <div class="header-controls">
                <form action="/" method="get" style="margin: 0;">
                    <select name="file" onchange="this.form.submit()" style="padding: 5px;">
                        {{range .AvailableResultFiles}}
                            <option value="{{.}}" {{if eq . $.LatestResultsFile}}selected{{end}}>{{.}}</option>
                        {{end}}
                    </select>
                </form>
                <button id="view-helix-yaml" onclick="viewHelixYaml()">View helix.yaml</button>
            </div>
        </div>
        <div class="results-container">
            <table>
                <thead>
                    <tr>
                        <th>Test Name</th>
                        <th>Result</th>
                        <th>Reason</th>
                        <th>Model</th>
                        <th>Inference Time</th>
                        <th>Evaluation Time</th>
                        <th>Session Link</th>
                        <th>Debug Link</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Tests}}
                    <tr class="{{if eq .Result "PASS"}}pass{{else}}fail{{end}}">
                        <td>{{.TestName}}</td>
                        <td>{{.Result}}</td>
                        <td class="truncate" data-full-text="{{.Reason}}">{{truncate .Reason 100}}</td>
                        <td>{{.Model}}</td>
                        <td>{{printf "%.2f" .InferenceTime.Seconds}}s</td>
                        <td>{{printf "%.2f" .EvaluationTime.Seconds}}s</td>
                        <td><a href="#" onclick="openDashboard('https://app.tryhelix.ai/session/{{.SessionID}}'); return false;">Session</a></td>
                        <td><a href="#" onclick="openDashboard('https://app.tryhelix.ai/dashboard?tab=llm_calls&filter_sessions={{.SessionID}}'); return false;">Debug</a></td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>
    <div id="iframe-container">
        <div id="resize-handle"></div>
        <div id="close-iframe" onclick="closeDashboard()">Close</div>
        <iframe id="dashboard-iframe" src=""></iframe>
    </div>
    <div id="tooltip" class="tooltip"></div>
    <script>
        function openDashboard(url) {
            document.getElementById('dashboard-iframe').src = url;
            document.getElementById('iframe-container').style.display = 'block';
            adjustContentHeight();
        }
        function closeDashboard() {
            document.getElementById('iframe-container').style.display = 'none';
            document.getElementById('dashboard-iframe').src = '';
            adjustContentHeight();
        }

        function adjustContentHeight() {
            const mainContainer = document.querySelector('.main-container');
            const iframeContainer = document.getElementById('iframe-container');
            if (iframeContainer.style.display === 'block') {
                mainContainer.style.height = 'calc(100vh - ' + iframeContainer.offsetHeight + 'px)';
            } else {
                mainContainer.style.height = '100vh';
            }
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
            adjustContentHeight();
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

        // Tooltip functionality
        const tooltip = document.getElementById('tooltip');
        document.querySelectorAll('.truncate').forEach(el => {
            el.addEventListener('mouseover', function(e) {
                tooltip.textContent = this.getAttribute('data-full-text');
                tooltip.style.display = 'block';
                tooltip.style.left = e.pageX + 'px';
                tooltip.style.top = e.pageY + 'px';
            });
            el.addEventListener('mouseout', function() {
                tooltip.style.display = 'none';
            });
        });

        // Initial adjustment
        adjustContentHeight();
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

	funcMap := template.FuncMap{
		"truncate": truncate,
	}

	tmpl, err := template.New("results").Funcs(funcMap).Parse(htmlTemplate)
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
