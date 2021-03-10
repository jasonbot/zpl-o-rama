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
