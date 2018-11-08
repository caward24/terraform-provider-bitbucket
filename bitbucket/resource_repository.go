package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

type CloneUrl struct {
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
}

type Repository struct {
	SCM         string `json:"scmId,omitempty"`
	Forkable  	bool   `json:"forkable,omitempty"`
	Name        string `json:"name,omitempty"`
	Slug        string `json:"slug,omitempty"`
	ID        	string `json:"id,omitempty"`
	Origin    	struct {
		Project struct {
			Key 		string `json:"key,omitempty"`
			Description string `json:"description,omitempty"`
		} `json:"project,omitempty"`
		Public   		bool   `json:"public,omitempty"`
		Links 	struct {
			Clone []CloneUrl   `json:"clone,omitempty"`
		} `json:"links,omitempty"`
	} `json:"origin,omitempty"`
}

func resourceRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryCreate,
		Update: resourceRepositoryUpdate,
		Read:   resourceRepositoryRead,
		Delete: resourceRepositoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"scmId": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "git",
			},
			"project_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"public": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"forkable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"slug": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func newRepositoryFromResource(d *schema.ResourceData) *Repository {
	repo := &Repository{
		Name:      	d.Get("name").(string),
		Slug:       d.Get("slug").(string),
		Forkable:  	d.Get("forkable").(bool),
		SCM:        d.Get("scmId").(string),
		ID:         d.Get("id").(string),
	}
	repo.Origin.Project.Description  = 	d.Get("description").(string)
	repo.Origin.Project.Key          = 	d.Get("project_key").(string)
	repo.Origin.Public               =	d.Get("public").(bool)

	return repo
}

func resourceRepositoryUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	repository := newRepositoryFromResource(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(repository)

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}

	_, err := client.Put(fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s",
		d.Get("project_key").(string),
		repoSlug,
	), jsonpayload)

	if err != nil {
		return err
	}

	return resourceRepositoryRead(d, m)
}

func resourceRepositoryCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	repo := newRepositoryFromResource(d)

	bytedata, err := json.Marshal(repo)

	if err != nil {
		return err
	}

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}

	_, err = client.Post(fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s",
		d.Get("project_key").(string),
		repoSlug,
	), bytes.NewBuffer(bytedata))

	if err != nil {
		return err
	}

	d.SetId(string(fmt.Sprintf("%s/%s", d.Get("project_key").(string), repoSlug)))

	return resourceRepositoryRead(d, m)
}
func resourceRepositoryRead(d *schema.ResourceData, m interface{}) error {
	id := d.Id()
	if id != "" {
		idparts := strings.Split(id, "/")
		if len(idparts) == 2 {
			d.Set("key", idparts[0])
			d.Set("slug", idparts[1])
		} else {
			return fmt.Errorf("Incorrect ID format, should match `project_key/slug`")
		}
	}

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}

	client := m.(*BitbucketClient)
	repo_req, _ := client.Get(fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s",
		d.Get("project_key").(string),
		repoSlug,
	))

	if repo_req.StatusCode == 200 {

		var repo Repository

		body, readerr := ioutil.ReadAll(repo_req.Body)
		if readerr != nil {
			return readerr
		}

		decodeerr := json.Unmarshal(body, &repo)
		if decodeerr != nil {
			return decodeerr
		}

		d.Set("scmId", repo.SCM)
		d.Set("public", repo.Origin.Public)
		d.Set("name", repo.Name)
		if repo.Slug != "" && repo.Name != repo.Slug {
			d.Set("slug", repo.Slug)
		}
		d.Set("forkable", repo.Forkable)
		d.Set("description", repo.Origin.Project.Description)
		d.Set("project_key", repo.Origin.Project.Key)

		for _, clone_url := range repo.Origin.Links.Clone {
			if clone_url.Name == "https" {
				d.Set("clone_https", clone_url.Href)
			} else {
				d.Set("clone_ssh", clone_url.Href)
			}
		}
	}

	return nil
}

func resourceRepositoryDelete(d *schema.ResourceData, m interface{}) error {

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}

	client := m.(*BitbucketClient)
	_, err := client.Delete(fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s",
		d.Get("project_key").(string),
		repoSlug,
	))

	return err
}
