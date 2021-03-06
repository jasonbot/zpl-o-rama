function onSignIn(googleUser) {
  var profile = googleUser.getBasicProfile();

  window.googleUser = googleUser;

  const idToken = googleUser.getAuthResponse().id_token;
  localStorage.setItem("profile-token", idToken);

  var auth2 = gapi.auth2.getAuthInstance();
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

        e.json().then((j) => window.location.reload());
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
  gapi.signin2.render('my-signin2', {
    'scope': 'profile email',
    'width': 100,
    'height': 20,
    'longtitle': false,
    'theme': 'light',
    'onsuccess': onSignIn,
    'onfailure': onFailure
  });
}

function handleHotwireResponse(r) {
  document.getElementById(r.div_id).innerHTML = r.HTML;
}
