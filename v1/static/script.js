function onSignIn(googleUser) {
  var profile = googleUser.getBasicProfile();

  window.googleUser = googleUser;

  const idToken = googleUser.getAuthResponse().id_token;
  localStorage.setItem("profile-token", idToken);

  var auth2 = gapi.auth2.getAuthInstance();
  handleHotwireResponse({
    div_id: "mainsection",
    HTML: "Finishing login flow...",
  });

  auth2.signOut().then(() => {
    fetch("/login", {
      method: "POST",
      body: JSON.stringify({ id_token: idToken }),
      headers: { "Content-Type": "application/json" },
    }).then((e) => {
      if (e.ok) {
        localStorage.setItem("profile-signedin", "true");
        localStorage.setItem("profile-id", profile.getId());
        localStorage.setItem("profile-name", profile.getName());
        localStorage.setItem("profile-image", profile.getImageUrl());
        localStorage.setItem("profile-email", profile.getEmail());
        localStorage.removeItem("profile-token");

        window.location.reload();
      } else {
        handleHotwireResponse({
          div_id: "mainsection",
          HTML: "There was an error logging in! Maybe you're not allowed to?",
        });
      }
    });
  });
}

function signOut() {
  fetch("/logout", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  }).then((e) => {
    localStorage.setItem("profile-signedin", "false");
    localStorage.removeItem("profile-id");
    localStorage.removeItem("profile-name");
    localStorage.removeItem("profile-image");
    localStorage.removeItem("profile-email");

    if (e.ok) {
      window.location.reload();
    }
  });
}

function onFailure(e) {}
function renderButton() {
  gapi.signin2.render("app-signin", {
    scope: "profile email",
    width: 100,
    height: 20,
    longtitle: false,
    theme: "light",
    onsuccess: onSignIn,
    onfailure: onFailure,
  });
}

function handleHotwireResponse(r) {
  if (!!r.div_id) {
    document.getElementById(r.div_id).innerHTML = r.HTML;
  }

  if (!!r.areas) {
    for (const [key, value] of Object.entries(r.areas)) {
      document.getElementById(key).innerHTML = value;
    }
  }
}

function updateJobStatus(jobid) {
  fetch(`/job/${jobid}/partial`).then((e) => {
    if (e.ok) {
      e.json().then((j) => { 
        handleHotwireResponse(j);

        if (j.message == "PENDING" || j.message == "PROCESSING") {
          window.setTimeout(() => updateJobStatus(jobid), 1000)
        }
      })
    }
  })
}
