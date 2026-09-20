package http

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/nodelink"
)

// runNodeLink is the secondary's side of the link: it keeps one outbound connection to the primary
// and answers the primary's requests on it directly with this server's own router, so nothing has to
// listen for them. It reconnects by itself, so a primary restart or a dropped connection heals.
func (s *Server) runNodeLink() {
	url := strings.TrimSuffix(s.config.Node.PrimaryNodeURL, "/") + constants.LinkPath
	header := http.Header{
		constants.HeaderLinkNodeID:   {s.config.Node.ID},
		constants.HeaderLinkNodeName: {s.config.Node.Name},
		constants.HeaderLinkNodeKey:  {s.config.Node.APIKey},
	}
	// Only a new node needs a token. A known node is recognised by its key, so leaving the (spent)
	// token configured does no harm.
	if s.config.Node.RegistrationToken != "" {
		header.Set(constants.HeaderLinkJoinToken, s.config.Node.RegistrationToken)
	}
	if strings.HasPrefix(url, "http://") {
		slog.Warn("the node link goes over plain HTTP: the node key is sent unencrypted. Use an https PRIMARY_NODE_URL outside a trusted network")
	}
	slog.Info("starting the outbound node link", "primary", s.config.Node.PrimaryNodeURL)

	(&nodelink.Client{
		URL:     url,
		Header:  header,
		Handler: s.engine,
		Logger:  slog.Default(),
	}).Run(s.shutdownCtx)
}
