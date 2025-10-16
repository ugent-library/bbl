package urls

import (
	"fmt"
	"net/url"

	"github.com/ugent-library/bbl"
)

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

func BackofficeOrganizations() string {
	return "/backoffice/organizations"
}

func BackofficePeople() string {
	return "/backoffice/people"
}

func BackofficeProjects() string {
	return "/backoffice/projects"
}

func BackofficeWorks(scope string, opts *bbl.SearchOpts) string {
	params := url.Values{}
	if scope != "" {
		params.Set("scope", scope)
	}
	if opts != nil {
		if opts.Query != "" {
			params.Add("q", opts.Query)
		}
		if opts.Size != 0 {
			params.Add("size", fmt.Sprint(opts.Size))
		}
		if opts.From != 0 {
			params.Add("from", fmt.Sprint(opts.From))
		}
		if opts.Cursor != "" {
			params.Add("cursor", fmt.Sprint(opts.From))
		}
	}
	return "/backoffice/works"
}

func BackofficeExportWorks(format string) string {
	return "/backoffice/works/export/" + url.PathEscape(format)
}

func BackofficeBatchEditWorks() string {
	return "/backoffice/works/batch_edit"
}

func BackofficeWork(id string) string {
	return "/backoffice/work/" + url.PathEscape(id)
}

func BackofficeWorkChanges(id string) string {
	return "/backoffice/work/" + url.PathEscape(id) + "/changes"
}

func BackofficeAddWorks() string {
	return "/backoffice/works/add"
}

func BackofficeCreateWork() string {
	return "/backoffice/works"
}

func BackofficeEditWork(id string) string {
	return "/backoffice/works/" + url.PathEscape(id) + "/edit"
}

func BackofficeWorkChangeKind() string {
	return "/backoffice/works/_change_kind"
}

func BackofficeWorkAddTitle() string {
	return "/backoffice/works/_add_title"
}

func BackofficeWorkRemoveTitle() string {
	return "/backoffice/works/_remove_title"
}

func BackofficeWorkAddAbstract() string {
	return "/backoffice/works/_add_abstract"
}

func BackofficeWorkEditAbstract() string {
	return "/backoffice/works/_edit_abstract"
}

func BackofficeWorkRemoveAbstract() string {
	return "/backoffice/works/_remove_abstract"
}

func BackofficeWorkAddLaySummary() string {
	return "/backoffice/works/_add_lay_summary"
}

func BackofficeWorkEditLaySummary() string {
	return "/backoffice/works/_edit_lay_summary"
}

func BackofficeWorkRemoveLaySummary() string {
	return "/backoffice/works/_remove_lay_summary"
}

func BackofficeWorkAddIdentifier() string {
	return "/backoffice/works/_add_identifier"
}

func BackofficeWorkRemoveIdentifier() string {
	return "/backoffice/works/_remove_identifier"
}

func BackofficeWorkAddContributor() string {
	return "/backoffice/works/_add_contributor"
}

func BackofficeWorkSuggestContributor() string {
	return "/backoffice/works/_suggest_contributor"
}

func BackofficeWorkEditContributor() string {
	return "/backoffice/works/_edit_contributor"
}

func BackofficeWorkRemoveContributor() string {
	return "/backoffice/works/_remove_contributor"
}

func BackofficeWorkAddFiles() string {
	return "/backoffice/works/_add_files"
}

func BackofficeWorkRemoveFile() string {
	return "/backoffice/works/_remove_file"
}

func BackofficeFileUploadURL() string {
	return "/backoffice/files/upload_url"
}
