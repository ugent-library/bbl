// Code generated by templ - DO NOT EDIT.

// templ: version: v0.3.833
package workviews

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import (
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views"
	"github.com/ugent-library/bbl/app/views/forms"
)

func Edit(c views.Ctx, rec *bbl.Work, formProfile *forms.Profile) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		templ_7745c5c3_Var2 := templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
			templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
			templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
			if !templ_7745c5c3_IsBuffer {
				defer func() {
					templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
					if templ_7745c5c3_Err == nil {
						templ_7745c5c3_Err = templ_7745c5c3_BufErr
					}
				}()
			}
			ctx = templ.InitializeContext(ctx)
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 1, "<div class=\"w-100 h-100 d-flex flex-column overflow-hidden\"><div class=\"bc-navbar bc-navbar--white bc-navbar--auto bc-navbar--bordered-bottom flex-column align-items-start\"><div class=\"bc-toolbar bc-toolbar-sm-responsive w-100\"><div class=\"bc-toolbar-left mb-1\"><div class=\"d-inline-flex align-items-center flex-wrap\"><span data-bbl-target=\"work-summary-status\"></span> <span class=\"c-subline text-nowrap me-3 pe-3 border-end\" data-bbl-target=\"work-summary-kind\"></span></div></div><div class=\"bc-toolbar-right mb-3 mb-lg-0\"><div class=\"bc-toolbar-item ps-0 ps-lg-4\"><div class=\"c-button-toolbar\"><button class=\"btn\" data-bbl-trigger=\"save-work\">Save</button> <a class=\"btn btn-success\" href=\"#\">Publish to Biblio</a><div class=\"dropdown\"><button class=\"btn btn-outline-secondary btn-icon-only me-0\" type=\"button\" data-bs-toggle=\"dropdown\" aria-haspopup=\"true\" aria-expanded=\"false\"><i class=\"if if-more\"></i><div class=\"visually-hidden\">More options</div></button><div class=\"dropdown-menu\" style=\"\"><button class=\"dropdown-item\" type=\"button\" data-bs-toggle=\"modal\" data-bs-target=\"#delete\"><i class=\"if if-delete\"></i> <span>Delete</span></button></div></div></div></div></div></div><h4 class=\"w-100 c-body-small mb-4\" data-bbl-target=\"work-summary-title\"></h4><div class=\"bc-toolbar flex-column flex-md-row align-items-start pb-4 h-auto\"><div class=\"bc-toolbar-left mt-3 mt-md-0\" data-bbl-target=\"work-summary-id\"></div></div></div>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = editForm(c, rec, formProfile).Render(ctx, templ_7745c5c3_Buffer)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 2, "</div>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			return nil
		})
		templ_7745c5c3_Err = views.Page(c, "Edit").Render(templ.WithChildren(ctx, templ_7745c5c3_Var2), templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

func editForm(c views.Ctx, rec *bbl.Work, formProfile *forms.Profile) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var3 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var3 == nil {
			templ_7745c5c3_Var3 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 3, "<div data-bbl-target=\"work-edit\" class=\"d-flex flex-grow-1 flex-shrink-1 overflow-hidden position-relative\"><div class=\"c-sub-sidebar c-sub-sidebar--responsive h-100 u-z-reset d-none d-lg-block\"><div class=\"bc-navbar bc-navbar--large\"><div class=\"bc-toolbar\"><div class=\"bc-toolbar-left\"><div class=\"bc-toolbar-item\"><h4 class=\"bc-toolbar-title\">Sidebar</h4></div></div></div></div><div class=\"c-sub-sidebar__content pt-5\"><div class=\"ps-6\"><nav class=\"nav nav-pills flex-column\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		for _, section := range formProfile.Sections {
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 4, "<a class=\"nav-link\" href=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var4 templ.SafeURL = section.Anchor()
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(string(templ_7745c5c3_Var4)))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 5, "\"><span class=\"c-sidebar__label\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var5 string
			templ_7745c5c3_Var5, templ_7745c5c3_Err = templ.JoinStringErrs(section.Name)
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `app/views/works/edit.templ`, Line: 77, Col: 53}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var5))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 6, "</span></a>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 7, "</nav></div></div></div><div class=\"w-100 u-scroll-wrapper\"><div class=\"u-scroll-wrapper__body w-100 p-6\"><form hx-encoding=\"multipart/form-data\" hx-post=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var6 string
		templ_7745c5c3_Var6, templ_7745c5c3_Err = templ.JoinStringErrs(c.Route("update_work", "work_id", rec.ID).String())
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `app/views/works/edit.templ`, Line: 88, Col: 65}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var6))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 8, "\" hx-target=\"[data-bbl-target=work-edit]\" hx-swap=\"outerHTML\" hx-trigger=\"click from:[data-bbl-trigger=save-work]\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		for _, section := range formProfile.Sections {
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 9, "<div class=\"mb-6\" id=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var7 string
			templ_7745c5c3_Var7, templ_7745c5c3_Err = templ.JoinStringErrs(section.ID())
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `app/views/works/edit.templ`, Line: 95, Col: 41}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var7))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 10, "\"><div class=\"mb-4\"><h2>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var8 string
			templ_7745c5c3_Var8, templ_7745c5c3_Err = templ.JoinStringErrs(section.Name)
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `app/views/works/edit.templ`, Line: 97, Col: 26}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var8))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 11, "</h2></div><div class=\"card mb-6\"><div class=\"card-body\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			for _, field := range section.Fields {
				switch field.Field {
				case "classifications":
				case "conference":
					templ_7745c5c3_Err = conferenceField(c, rec).Render(ctx, templ_7745c5c3_Buffer)
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
				case "identifiers":
					templ_7745c5c3_Err = identifiersField(c, rec).Render(ctx, templ_7745c5c3_Buffer)
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
				case "keywords":
				case "kind":
				case "titles":
				}
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 12, "</div></div></div>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 13, "</form></div></div></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

func RefreshEditForm(c views.Ctx, rec *bbl.Work, formProfile *forms.Profile) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var9 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var9 == nil {
			templ_7745c5c3_Var9 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		templ_7745c5c3_Err = editForm(c, rec, formProfile).Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

//	templ classificationsField(c views.Ctx, rec *biblio.Work, only []string) {
//		for _, p := range filterSchemes(rec.Profile.Classifications.Schemes, only) {
//			if p.Multiple {
//				@forms.TextInputRepeat(forms.TextInputRepeatArgs{
//					FieldArgs: forms.FieldArgs{
//						Label:    p.Scheme,
//						Name:     fmt.Sprintf("Classifications[%s]", p.Scheme),
//						Required: p.Required,
//					},
//					Values: rec.Classifications.ValuesFor(p.Scheme),
//				})
//			} else {
//				@forms.TextInput(forms.TextInputArgs{
//					FieldArgs: forms.FieldArgs{
//						Label:    p.Scheme,
//						Name:     fmt.Sprintf("Classifications[%s]", p.Scheme),
//						Required: p.Required,
//					},
//					Value: rec.Classifications.ValueFor(p.Scheme),
//				})
//			}
//		}
//	}
func identifiersField(c views.Ctx, rec *bbl.Work) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var10 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var10 == nil {
			templ_7745c5c3_Var10 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		templ_7745c5c3_Err = forms.CodeAttrRepeat(forms.CodeAttrRepeatArgs{
			FieldArgs: forms.FieldArgs{
				Name: "identifiers",
			},
			Schemes: rec.Spec.Attrs["identifier"].Schemes,
			Attrs:   rec.Identifiers,
		}).Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

func conferenceField(c views.Ctx, rec *bbl.Work) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var11 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var11 == nil {
			templ_7745c5c3_Var11 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		if rec.Spec.Attrs["conference"].Use {
			val := rec.Conference.GetVal()
			templ_7745c5c3_Err = forms.TextInput(forms.TextInputArgs{
				FieldArgs: forms.FieldArgs{
					Label: "Conference",
					Name:  "conference.name",
				},
				Value: val.Name,
			}).Render(ctx, templ_7745c5c3_Buffer)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 14, " ")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = forms.TextInput(forms.TextInputArgs{
				FieldArgs: forms.FieldArgs{
					Label: "Conference location",
					Name:  "conference.location",
				},
				Value: val.Location,
			}).Render(ctx, templ_7745c5c3_Buffer)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 15, " ")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = forms.TextInput(forms.TextInputArgs{
				FieldArgs: forms.FieldArgs{
					Label: "Conference organizer",
					Name:  "conference.organizer",
				},
				Value: val.Organizer,
			}).Render(ctx, templ_7745c5c3_Buffer)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		return nil
	})
}

// templ keywordsField(c views.Ctx, rec *biblio.Work) {
// 	if rec.Profile.Keywords.Use {
// 		@forms.Tags(forms.TagsArgs{
// 			FieldArgs: forms.FieldArgs{
// 				Label:    "Keywords",
// 				Name:     "Keywords",
// 				Required: rec.Profile.Keywords.Required,
// 			},
// 			Values: rec.Keywords.Values(),
// 		})
// 	}
// }

// templ kindField(c views.Ctx, rec *biblio.Work) {
// 	<div class="form-group">
// 		<label class="form-label form-label-top">Work type</label>
// 		<select
// 			class="form-select"
// 			name="Kind"
// 			if rec.IsNew() {
// 				hx-post={ c.Route("refresh_new_work").String() }
// 			}
// 			hx-include="closest form"
// 			hx-target="[data-bbl-target=work-edit]"
// 			hx-swap="outerHTML"
// 		>
// 			<option></option>
// 			for _, profile := range biblio.WorkProfiles() {
// 				<option value={ profile.Kind } selected?={ rec.Kind == profile.Kind }>{ profile.Kind }</option>
// 			}
// 		</select>
// 	</div>
// }

// templ titlesField(c views.Ctx, rec *biblio.Work) {
// 	if rec.Profile.Titles.Use {
// 		@forms.TextRepeat(forms.TextRepeatArgs{
// 			FieldArgs: forms.FieldArgs{
// 				Label:    "Titles",
// 				Name:     "Titles",
// 				Required: rec.Profile.Titles.Required,
// 			},
// 			Values: rec.Titles,
// 		})
// 	}
// }

//	func filterSchemes(schemes []bbl.WorkProfileScheme, only []string) []bbl.WorkProfileScheme {
//		if len(only) == 0 {
//			return schemes
//		}
//		return lo.Filter(schemes, func(s bbl.WorkProfileScheme, _ int) bool {
//			return slices.Contains(only, s.Scheme)
//		})
//	}
var _ = templruntime.GeneratedTemplate
