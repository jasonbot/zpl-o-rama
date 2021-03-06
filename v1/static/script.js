function onSignIn(googleUser) {
  var profile = googleUser.getBasicProfile();
  console.log("ID: " + profile.getId());
  console.log("Name: " + profile.getName());
  console.log("Image URL: " + profile.getImageUrl());
  console.log("Email: " + profile.getEmail());

  window.googleUser = googleUser;

  console.log(googleUser.getAuthResponse().id_token);

  const idToken = googleUser.getAuthResponse().id_token;
  localStorage.setItem("profile-token", idToken);

  fetch("/login", {
    method: "POST",
    body: JSON.stringify({ id_token: idToken }),
  }).then((e) => {
    if (e.ok) {
      localStorage.setItem("profile-signedin", "true");
      localStorage.setItem("profile-id", profile.getId());
      localStorage.setItem("profile-name", profile.getName());
      localStorage.setItem("profile-image", profile.getImageUrl());
      localStorage.setItem("profile-email", profile.getEmail());
      localStorage.removeItem("profile-token");

      e.json().then((j) => handleHotwireResponse(e));
    }
  });
}

function signOut() {
  fetch("/logout", { method: "POST" }).then((e) => {
    localStorage.setItem("profile-signedin", "false");
    localStorage.removeItem("profile-id");
    localStorage.removeItem("profile-name");
    localStorage.removeItem("profile-image");
    localStorage.removeItem("profile-email");
    console.log("OUT");

    if (e.ok) {
      e.json().then((j) => handleHotwireResponse(e));
    }
  });

  if (gapi !== undefined && gapi.auth2 !== undefined) {
    var auth2 = gapi.auth2.getAuthInstance();
    auth2.signOut();
  }
}

function handleHotwireResponse(r) {
  document.getElementById(r.div_id).innerHTML = r.HTML;
}
