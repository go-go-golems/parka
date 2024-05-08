package render

import (
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"html/template"
	"net/http"
	"strings"
)

// Renderer is a struct that is able to lookup a page name and render it.
// It can handle a variety of formats:
// - static HTML
// - static markdown
// - templated HTML
// - templated markdown
// - base HTML template to render markdown into
// - index page templates
//
// It is an early replacement for what used to be the templateLookup approach in parka.Server.
//
// # NOTE(manuel, 2023-05-26) Some notes on the implementation rationale
//
// Driven by the needs for richer handling of templates mixing commands and content when building
// reports pages with sqleton, the templateLookup approach of server, which was confusing from the start,
// was breaking down.
//
// In order to more easily to configure the reports page, which would expose different command repositories
// with manually overloaded rendering templates (datatables.tmpl.html) as well as static content (index page,
// documentation pages), I refactored the haphazard templatelookup concept to be configured from a config file
// and split it into different "modules" that could be "mounted" under different paths:
// - static file
// - static dir
// - single command
// - command dir
// - template file
// - template dir
//
// It is while implementing the template dir that I realized that a more generic "render paths based on configured templates
// directories and data" as a common building block. In a way it is what I was originally searching for with the template
// lookups pattern, but I mixed it with the concerns of serving files in parka, instead of just focusing on "render pages".
type Renderer struct {
	Data            map[string]interface{}
	TemplateLookups []TemplateLookup

	// MarkdownBaseTemplateName is used to wrap markdown that was rendered into HTML into a top-level page
	// NOTE(manuel, 2023-05-26) Maybe this should be a templateLookup that always returns the same template?
	MarkdownBaseTemplateName string

	IndexTemplateName string
}

type RendererOption func(r *Renderer) error

// WithReplaceData will replace the data the renderer uses to render out templates with the passed in data.
func WithReplaceData(data map[string]interface{}) RendererOption {
	return func(r *Renderer) error {
		r.Data = data
		return nil
	}
}

// WithMergeData will merge the given data into the renderer's data.
func WithMergeData(data map[string]interface{}) RendererOption {
	return func(r *Renderer) error {
		for k, v := range data {
			r.Data[k] = v
		}
		return nil
	}
}

// WithMarkdownBaseTemplateName will set the name of the template to use as the base template when rendering
// the HTML coming out of markdown rendering into HTML.
func WithMarkdownBaseTemplateName(name string) RendererOption {
	return func(r *Renderer) error {
		r.MarkdownBaseTemplateName = name
		return nil
	}
}

// WithPrependTemplateLookups will prepend the given template lookups to the list of lookups,
// ensuring that they will be found before whatever templates might already be in the list.
func WithPrependTemplateLookups(lookups ...TemplateLookup) RendererOption {
	return func(s *Renderer) error {
		// prepend lookups to the list
		s.TemplateLookups = append(lookups, s.TemplateLookups...)
		return nil
	}
}

// WithAppendTemplateLookups will append the given template lookups to the list of lookups,
// but they will be found after whatever templates might already be in the list. This is great
// for providing fallback templates.
func WithAppendTemplateLookups(lookups ...TemplateLookup) RendererOption {
	return func(s *Renderer) error {
		// append lookups to the list
		s.TemplateLookups = append(s.TemplateLookups, lookups...)
		return nil
	}
}

func WithIndexTemplateName(name string) RendererOption {
	return func(r *Renderer) error {
		r.IndexTemplateName = name
		return nil
	}
}

// WithReplaceTemplateLookups will replace any existing template lookups with the given ones.
func WithReplaceTemplateLookups(lookups ...TemplateLookup) RendererOption {
	return func(s *Renderer) error {
		s.TemplateLookups = lookups
		return nil
	}
}

func NewRenderer(opts ...RendererOption) (*Renderer, error) {
	r := &Renderer{
		IndexTemplateName: "index",
	}

	for _, opt := range opts {
		err := opt(r)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

// LookupTemplate will iterate through the template lookups until it finds one of the
// templates given in name.
func (r *Renderer) LookupTemplate(name ...string) (*template.Template, error) {
	var t *template.Template

	for _, lookup := range r.TemplateLookups {
		t, err := lookup.Lookup(name...)
		if err != nil {
			log.Debug().Err(err).Strs("name", name).Msg("failed to lookup template, skipping")

		}
		if err == nil {
			return t, nil
		}
	}

	return t, nil
}

// Render a given page with the given data.
//
// It first looks for a markdown file or template called either page.md or page.tmpl.md,
// and render it as a template, passing it the given data.
// It will use base.tmpl.html as the base template for serving the resulting markdown HTML.
// page.md is rendered as a plain markdown file, while page.tmpl.md is rendered as a template.
//
// If no markdown file or template is found, it will look for a HTML file or template called
// either page.html or page.tmpl.html and serve it as a template, passing it the given data.
// page.html is served as a plain HTML file, while page.tmpl.html is served as a template.
//
// NOTE(manuel, 2023-06-21)
// Render renders directly into the http.ResponseWriter, which means that an error in the template
// rendering will not be able to update the headers, as those will have already been sent.
// This could lead to partial writes with an error code of 200 if there is an error rendering the template,
// not sure if that's exactly what we want.
func (r *Renderer) Render(
	c echo.Context,
	page string,
	data map[string]interface{},
) error {
	// first, merge the data we want to pass to the templates, with the data passed in overridding
	// the data in the renderer
	data_ := map[string]interface{}{}
	for k, v := range r.Data {
		data_[k] = v
	}
	for k, v := range data {
		data_[k] = v
	}

	// TODO(manuel, 2023-05-26) Don't render plain files as templates
	// See https://github.com/go-go-golems/parka/issues/47
	t, err := r.LookupTemplate(page+".tmpl.md", page+".md", page)
	if err != nil {
		return errors.Wrap(err, "error looking up template")
	}

	baseTemplate, err := r.LookupTemplate(r.MarkdownBaseTemplateName)
	if err != nil {
		return errors.Wrap(err, "error looking up base template")
	}

	if baseTemplate == nil {
		// no base template to render the markdown to HTML, so just return the markdown
		c.Response().Header().Set("Content-Type", "text/plain")
		c.Response().WriteHeader(http.StatusOK)

		err := t.Execute(c.Response(), data)
		if err != nil {
			return errors.Wrap(err, "error executing template")
		}

		return nil
	}

	if t != nil {
		markdown, err := RenderMarkdownTemplateToHTML(t, nil)
		if err != nil {
			return errors.Wrap(err, "error rendering markdown")
		}

		c.Response().WriteHeader(http.StatusOK)
		err = baseTemplate.Execute(
			c.Response(),
			map[string]interface{}{
				"markdown": template.HTML(markdown),
			})
		if err != nil {
			return errors.Wrap(err, "error executing base template")
		}
	} else {
		t, err = r.LookupTemplate(page+".tmpl.html", page+".html")
		if err != nil {
			return errors.Wrap(err, "error looking up template")
		}
		if t == nil {
			return &NoPageFoundError{Page: page}
		}

		c.Response().WriteHeader(http.StatusOK)

		err := t.Execute(c.Response(), data)
		if err != nil {
			return errors.Wrap(err, "error executing template")
		}
	}
	return nil
}

func (r *Renderer) HandleWithTemplate(
	path string,
	templateName string,
	data map[string]interface{},
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Response().Committed {
				return next(c)
			}

			if c.Request().URL.Path == path {
				err := r.Render(c, templateName, data)
				if err != nil {
					if _, ok := err.(*NoPageFoundError); ok {
						return next(c)
					}
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				return nil
			}

			return next(c)
		}
	}
}

func (r *Renderer) HandleWithTrimPrefix(prefix string, data map[string]interface{}) echo.MiddlewareFunc {
	prefix = strings.TrimPrefix(prefix, "/")
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Response().Committed {
				return next(c)
			}

			rawPath := c.Request().URL.Path

			if len(rawPath) > 0 && rawPath[0] == '/' {
				trimmedPath := rawPath[1:]
				trimmedPath = strings.TrimPrefix(trimmedPath, prefix)
				if trimmedPath == "" || strings.HasSuffix(trimmedPath, "/") {
					trimmedPath += "index"
				}

				err := r.Render(c, trimmedPath, data)
				if err != nil {
					if _, ok := err.(*NoPageFoundError); ok {
						return next(c)
					}
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}

				return nil
			}

			// TODO(manuel, 2024-05-07) I'm not entirely sure this is the correct way of doing things
			// this is if the rawPath is empty? I'm not sure I understand the logic here
			return c.NoContent(http.StatusOK)
		}
	}
}
