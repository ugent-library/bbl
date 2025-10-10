package urls

import "net/url"

func Works() string {
	return "/works"
}

func Work(id string) string {
	return "/works/" + url.PathEscape(id)
}

func BackofficeHome() string {
	return "/backoffice"
}

func BackofficeLogin() string {
	return "/backoffice/login"
}

func BackofficeLogout() string {
	return "/backoffice/logout"
}

func BackofficeSSE(token string) string {
	return "/backoffice/sse?token=" + url.QueryEscape(token)
}

func BackofficeOrganizations() string {
	return "/backoffice/organizations"
}

func BackofficePeople() string {
	return "/backoffice/people"
}

func BackofficeProjects() string {
	return "/backoffice/projects"
}

func BackofficeWorks() string {
	return "/backoffice/works"
}

func BackofficeFileUploadURL() string {
	return "/backoffice/files/upload_url"
}
