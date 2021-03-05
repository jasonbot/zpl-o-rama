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

type printJobRequest struct {
	Jobid  string `json:"jobid,omitempty"`
	ZPL    string `json:"ZPL"`
	Author string `json:"author"`
}

type printJobStatus struct {
	Jobid    string        `json:"jobid"`
	Status   pictureStatus `json:"status"`
	ZPL      string        `json:"ZPL"`
	ImageB64 string        `json:"image"`
	Created  string        `json:"created"`
	Updated  string        `json:"updated"`
	Author   string        `json:"author"`
	Message  string        `json:"message"`
}

// Make this struct boltable
func (*printJobStatus) Table() string {
	return printjobTable
}

func (job *printJobStatus) Key() string {
	return job.Jobid
}
