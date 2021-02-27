function onSignIn(googleUser) {
    var profile = googleUser.getBasicProfile();
    console.log('ID: ' + profile.getId());
    console.log('Name: ' + profile.getName());
    console.log('Image URL: ' + profile.getImageUrl());
    console.log('Email: ' + profile.getEmail());

    window.googleUser = googleUser;

    console.log(googleUser.getAuthResponse().id_token)

    localStorage.setItem('profile-signedin', 'true');
    localStorage.setItem('profile-id', profile.getId());
    localStorage.setItem('profile-name', profile.getName());
    localStorage.setItem('profile-image', profile.getImageUrl());
    localStorage.setItem('profile-email', profile.getEmail());
    localStorage.setItem('profile-token', googleUser.getAuthResponse().id_token);
}

function signOut() {
    var auth2 = gapi.auth2.getAuthInstance();
    auth2.signOut().then(function () {
        localStorage.setItem('profile-signedin', 'false');
        console.log('User signed out.');
    });
}
