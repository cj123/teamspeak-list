package main

import (
	"embed"
	"html/template"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/multiplay/go-ts3"
	"github.com/sirupsen/logrus"
)

var (
	server   = os.Getenv("TS3_SERVER")
	port     = os.Getenv("TS3_PORT")
	username = os.Getenv("TS3_USERNAME")
	password = os.Getenv("TS3_PASSWORD")
	serverID = os.Getenv("TS3_SERVERID")
)

func main() {
	r := mux.NewRouter()
	updated := time.Now()

	var channels []*ts3.Channel
	var users []*ts3.OnlineClient
	var serverInfo *ts3.Server

	t, err := template.New("users").Funcs(map[string]interface{}{
		"yes": func(b bool) string {
			if b {
				return "yes"
			} else {
				return "no"
			}
		},
		"secondsToDuration": func(s int) string {
			return (time.Duration(s) * time.Second).String()
		},
		"add": func(i, j int) int {
			return i + j
		},
	}).Parse(htmlTemplate)

	if err != nil {
		panic(err)
	}

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := t.Execute(w, map[string]interface{}{
			"channels": channels,
			"users":    users,
			"updated":  updated.Format(time.RFC1123),
			"server":   serverInfo,
		})

		if err != nil {
			panic(err)
		}
	})

	go func() {
		for {
			server, updatedChannels, updatedUsers, err := update()

			if err != nil {
				logrus.Errorf("Unable to update, err: %s", err.Error())
				continue
			}

			channels = updatedChannels
			users = updatedUsers
			updated = time.Now()
			serverInfo = server

			time.Sleep(5 * time.Second)
		}
	}()

	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(content)))

	http.ListenAndServe("0.0.0.0:2208", r)
}

func update() (*ts3.Server, []*ts3.Channel, []*ts3.OnlineClient, error) {
	client, err := ts3.NewClient(server + ":" + port)

	if err != nil {
		return nil, nil, nil, err
	}

	defer client.Close()

	cmds := &ts3.ServerMethods{Client: client}

	u, _ := strconv.ParseInt(serverID, 0, 10)

	if err := cmds.Use(int(u)); err != nil {
		return nil, nil, nil, err
	}

	err = cmds.Login(username, password)

	if err != nil {
		return nil, nil, nil, err
	}

	serverInfo, err := cmds.Info()

	if err != nil {
		return nil, nil, nil, err
	}

	clientList, err := cmds.ClientList(ts3.ClientListFull)

	if err != nil {
		return nil, nil, nil, err
	}

	if len(clientList) == 0 {
		return nil, nil, nil, err
	}

	channels, err := cmds.ChannelList()

	if err != nil {
		return nil, nil, nil, err
	}

	var clientInfo []*ts3.OnlineClient

	for _, c := range clientList {
		if strings.Contains(*c.Platform, "ServerQuery") {
			continue
		}

		clientInfo = append(clientInfo, c)
	}

	sort.Slice(clientInfo, func(i, j int) bool {
		return clientInfo[i].Nickname < clientInfo[j].Nickname
	})

	return serverInfo, channels, clientInfo, nil
}

//go:embed html/index.html
var htmlTemplate string

//go:embed static
var content embed.FS
