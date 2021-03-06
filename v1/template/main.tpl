{{define "please-log-in"}}
    <p>Hi there. Welcome to ZPL-O-Matic.</p>
    <p>This site isn't very interesting unless you log in.</p>
{{end}}

{{define "input-zpl-form"}}
    <h1>Let's render some ZPL on physical media!</h1>
    <form action="/print" method="post">
        <div>
            <textarea id="zplinput" rows="15" cols="80" name="ZPL"></textarea>
        </div>
        <div>
            <button type="submit">Go do it</button>
        </div>
    </form>
{{end}}

{{define "loginbar"}}
    {{if ne .User ""}}
        Signed in as {{ html .User }}
        <a href="#" id="signout" onclick="signOut();">Sign out</a>
    {{else}}
        Not signed in
        <div id="my-signin2" data-onsuccess="onSignIn"></div>
    {{end}}
    <script src="https://apis.google.com/js/platform.js?onload=renderButton" async defer></script>
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
