package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type server struct {
	router     *gin.Engine
	httpclient *http.Client
}

type IServer interface {
	Start(port int) error
}

// New a proxy server
func New() IServer {
	r := gin.New()
	r.Use(gin.Logger())
	// r.Use(gin.Recovery())

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpclient := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // dont follow redirect
		},
	}

	return &server{
		router:     r,
		httpclient: httpclient,
	}
}

func (srv *server) Start(port int) error {
	srv.init()
	return srv.router.Run(fmt.Sprintf(":%d", port))
}

type QueryModel struct {
	Owner string `uri:"owner" binding:"required"`
	Repo  string `uri:"repo" binding:"required"`
	ID    string `uri:"id"`   // release id
	Name  string `uri:"name"` // asset name
	Tag   string `uri:"tag"`  // tag name
}

func (srv *server) init() {
	srv.router.GET("/:owner/:repo/releases/*any", queryEndpoint())
	srv.router.GET("/:owner/:repo/releases", queryEndpoint())
	// srv.router.GET("/:owner/:repo/download/:id/:name", downloadEndpoint())
	srv.router.GET("/:owner/:repo/download/tags/:tag/:name", downloadEndpoint(srv.httpclient))
}

func queryEndpoint() gin.HandlerFunc {
	endpoint := "https://api.github.com/repos"
	return func(c *gin.Context) {
		var params QueryModel
		if err := c.ShouldBindUri(&params); err != nil {
			c.JSON(400, gin.H{"msg": err})
			return
		}

		remote, err := url.Parse(endpoint)
		if err != nil {
			c.JSON(500, gin.H{"msg": err})
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		c.Request.Host = remote.Host
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func downloadEndpoint(httpclient *http.Client) gin.HandlerFunc {
	endpoint := "github.com"
	return func(c *gin.Context) {
		var params QueryModel
		if err := c.ShouldBindUri(&params); err != nil {
			c.JSON(400, gin.H{"msg": err})
			return
		}

		target := fmt.Sprintf("https://%s/%s/%s/releases/download/%s/%s",
			endpoint,
			params.Owner,
			params.Repo,
			params.Tag,
			params.Name,
		)
		request, _ := http.NewRequest("HEAD", target, nil)
		resp, err := httpclient.Do(request)
		if err != nil {
			c.JSON(502, gin.H{"msg": err})
			return
		}

		if resp.StatusCode != 302 {
			c.JSON(502, gin.H{"msg": "Could not get download url"})
			return
		}

		remote, _ := resp.Location()

		director := func(req *http.Request) {
			req.URL.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Path = remote.Path
			req.URL.RawQuery = remote.RawQuery
			req.Header.Set("Host", remote.Host)
			req.Host = req.URL.Host
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
