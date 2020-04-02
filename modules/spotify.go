package modules

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/olivia-ai/olivia/user"

	"golang.org/x/oauth2"

	"github.com/olivia-ai/olivia/language"
	"github.com/zmb3/spotify"
)

var (
	spotifySetterTag = "spotify setter"
	spotifyPlayerTag = "spotify player"

	tokenChannel = make(chan *oauth2.Token)
	state        = "abc123"
	auth         = spotify.NewAuthenticator(
		os.Getenv("CALLBACK_URL"),
		spotify.ScopeStreaming,
		spotify.ScopeUserModifyPlaybackState,
	)

	redirectURL = os.Getenv("REDIRECT_URL")

	loginMessage = `Login in progress <meta http-equiv="refresh" content="0; url = %s" />`
)

func init() {
	RegisterModule(Module{
		Tag: spotifySetterTag,
		Patterns: []string{
			"Here are my spotify tokens",
			"My spotify secrets",
		},
		Responses: []string{
			loginMessage,
		},
		Replacer: SpotifySetterReplacer,
	})

	RegisterModule(Module{
		Tag: spotifyPlayerTag,
		Patterns: []string{
			"Play from on Spotify",
		},
		Responses: []string{
			"Playing %s from %s on Spotify.",
		},
		Replacer: SpotifyPlayerReplacer,
	})
}

// SpotifySetterReplacer gets the tokens in the user entry and save them into the client's information.
// See modules/modules.go#Module.Replacer() for more details.
func SpotifySetterReplacer(entry, response, token string) (string, string) {
	spotifyTokens := language.SearchTokens(entry)

	// Returns if the token is empty
	if len(spotifyTokens) != 2 {
		return spotifySetterTag, "You need to send the two tokens."
	}

	// Generate the authentication url
	auth.SetAuthInfo(spotifyTokens[0], spotifyTokens[1])
	url := auth.AuthURL(state)

	// Waits for the authentication to be completed, and save the client in user's information
	go func() {
		authenticationToken := <-tokenChannel

		// Save the authentication token
		user.ChangeUserInformation(token, func(information user.Information) user.Information {
			information.SpotifyToken = authenticationToken

			return information
		})
	}()

	return spotifySetterTag, fmt.Sprintf(response, url)
}

// SpotifyPlayerReplacer plays a specified music on the user's spotify
// See modules/modules.go#Module.Replacer() for more details.
func SpotifyPlayerReplacer(entry, response, token string) (string, string) {
	authenticationToken := user.GetUserInformation(token).SpotifyToken
	client := auth.NewClient(authenticationToken)

	// Renew the authentication token
	if m, _ := time.ParseDuration("5m30s"); time.Until(authenticationToken.Expiry) < m {
		user.ChangeUserInformation(token, func(information user.Information) user.Information {
			information.SpotifyToken, _ = client.Token()
			return information
		})
	}

	music, artist := language.SearchMusic(entry)
	searchContent := music + " " + artist

	results, _ := client.Search(searchContent, spotify.SearchTypeTrack)

	// Return if no music was found
	if len(results.Tracks.Tracks) == 0 {
		return spotifyPlayerTag, "Sorry, no music was found."
	}

	track := results.Tracks.Tracks[0]

	// Play the found track
	client.PlayOpt(&spotify.PlayOptions{
		URIs: []spotify.URI{track.URI},
	})
	client.Play()

	return spotifyPlayerTag, fmt.Sprintf(response, track.Name, track.Artists[0].Name)
}

// CompleteAuth completes the Spotify authentication.
func CompleteAuth(w http.ResponseWriter, r *http.Request) {
	token, err := auth.Token(state, r)

	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}

	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		fmt.Printf("State mismatch: %s != %s\n", st, state)
	}

	// Use the token to get an authenticated client
	w.Header().Set("Content-Type", "text/html")
	// Redirect the user
	fmt.Fprintf(w, `<meta http-equiv="refresh" content="0; url = %s" />`, redirectURL)

	tokenChannel <- token
}
