package zplorama

const (
	printjobTable = "print-jobs"
	jobTimeTable  = "print-times"
)

// ConfStruct is the configuration for the services
type ConfStruct struct {
	GoogleSite        string   `json:"google_site"`
	FrontendPort      int      `json:"frontend_port"`
	PrintserviceHost  string   `json:"printservice_host"`
	PrintservicePort  int      `json:"printservice_port"`
	PrintTime         string   `json:"print_time"`
	AuthtokenLifetime string   `json:"authtoken_lifetime"`
	AuthSecret        string   `json:"authsecret"`
	AllowedLogins     []string `json:"allowed_logins"`
	BackendDatabase   string   `json:"backend_database"`
	FrontenedDatabase string   `json:"frontend_database"`
}

// Represents the state of the print job
type pictureStatus string

const (
	pending    pictureStatus = "PENDING"
	processing               = "PROCESSING"
	succeeded                = "SUCCEEDED"
	failed                   = "FAILED"
	missing                  = "MISSING"
)

const emptyPNG string = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="
const sadFace string = "iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAYAAADED76LAAAAQElEQVQY04WPSwrAQAxCnyH3v/LrpoU0DIxLP6gBUOWAJCnVJB8xRVSLC+qt+CUn19N9mtI7uc09Bu0HqOR28wH8uiIQ3tOhaQAAAABJRU5ErkJggg=="

type printJobRequest struct {
	ZPL    string `json:"ZPL" form:"ZPL" query:"ZPL"`
	Author string `json:"author"`
	// NOT PUBLIC -- assigned by the software at execution time
	jobid string
}

type printJobStatus struct {
	Jobid         string        `json:"jobid"`
	Status        pictureStatus `json:"status"`
	ZPL           string        `json:"ZPL"`
	ImageB64      string        `json:"image"`
	ImageB64Small string        `json:"image_small"`
	Created       string        `json:"created"`
	Updated       string        `json:"updated"`
	Author        string        `json:"author"`
	Message       string        `json:"message"`
	Log           []string      `json:"log"`
	Done          bool          `json:"done"`
}

// Make this struct boltable
func (*printJobStatus) Table() string {
	return printjobTable
}

func (job *printJobStatus) Key() string {
	return job.Jobid
}

type jobTimestamp struct {
	Timestamp string `json:"timestamp"`
	Jobid     string `json:"job_id"`
}

// Make this struct boltable
func (*jobTimestamp) Table() string {
	return jobTimeTable
}

func (ts *jobTimestamp) Key() string {
	return ts.Timestamp
}

type hotwireResponse struct {
	Message string            `json:"message"`
	Areas   map[string]string `json:"areas,omitempty"`
	DivID   string            `json:"div_id"`
	HTML    string            `json:"HTML"`
}

type errJSON struct {
	Errmsg string `json:"error"`
}
