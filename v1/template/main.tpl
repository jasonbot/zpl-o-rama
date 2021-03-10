{{define "please-log-in"}}
    <p>Hi there. Welcome to ZPL-O-Matic.</p>
    <p>This site isn't very interesting unless you log in.</p>
{{end}}

{{define "loginbar"}}
    {{if ne .User ""}}
        Signed in as {{ html .User }}
        <a href="#" id="signout" onclick="signOut();">Sign out</a>
    {{else}}
        Not signed in
        <a href="/signin">Sign in</a>
    {{end}}
{{end}}


{{define "input-zpl-form"}}
    <h1>Let's render some ZPL on physical media!</h1>
    <form action="/print" method="post">
        <div>
            <textarea name="ZPL" id="zplinput" rows="15" cols="80" name="ZPL"></textarea>
        </div>
        <div>
            <button type="submit" class="godoit">Go do it</button>
        </div>
    </form>
{{end}}

{{define "job-status-part"}}

    <h2>ID: <span id="jobid">{{ .Jobid }}</span></h2>
    <div>
        <p>Created <span id="jobcreated">{{ html .Created }}</span> by <span id="jobauthor">{{ html .Author }}</span></p>
        <p>
            <b>Job Status:</b> <span id="jobstatus" class="status-{{ html .Status }}">{{ html .Status }}</span>
            {{if not .Done }} <span class="spinner"></span> {{end}}
        </p>
    </div>
    <div id="zplimage" class="zplimage">
        {{ if .Done }} 
            <!-- ?{{ .Status }} is a cache-buster -->
            <img class="scanimage" src="/job/{{ .Jobid }}/image.png?{{ .Status }}" alt="Your image" />
            <p><a href="/job/{{ .Jobid }}/original.png" download>Download original size image</a></p>
        {{ end }} 

        {{ if eq .ZPL "" }}
            <p>(Note: A job submitted with empty ZPL just takes a picture)</p>
        {{ end }}
    </div>

    <h3>Job Run Log</h3>
    <div id="runlog">
        {{ range .Log }}
            <div>{{ html . }}</div>
        {{ end }}
    </div>
{{end}}

{{define "job-status"}}
    <div>
        <a href="/home">&larr; Back to home</a>
    </div>

    <h1>Job</h1>
    <div id="jobstatus">
        {{ template "job-status-part" . }}
    </div>

    {{if .Done }} <!-- Already done --> {{ else }} <script>updateJobStatus('{{ html .Jobid }}')</script> {{end}}

    {{ if ne .ZPL "" }}
        <hr />
        <h3>Original ZPL</h3>
        <pre>{{ html .ZPL }}</pre>
    {{ end }}
{{end}}

{{define "main"}}
    <!DOCTYPE html>
    <html lang="en">

    <head>
        <meta charset="utf-8">
        <meta name="google-signin-client_id" content="{{ GoogleSite }}">
        <title>{{ .Title }}</title>
        <link rel="stylesheet" href="/static/style.css">
        <script src="/static/script.js"></script>
    </head>

    <body>
        <div id="page">
            <div id="loginbar">
                {{ template "loginbar"  . }}
            </div>

            <div id="mainsection">
                {{ .Body }}
            </div>
        </div>
    </body>

    </html>
{{end}}
