package testplugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	c "go.minekube.com/common/minecraft/component"
	"net/http"
	"regexp"
	"strconv"
	"time"

	// "go.minekube.com/common/minecraft/color"
	// c "go.minekube.com/common/minecraft/component"
	. "github.com/minekube/gate-plugin-template/util"
	"go.minekube.com/gate/pkg/edition/java/proxy"
	"go.minekube.com/gate/pkg/edition/java/title"
)

type Permissionstruct struct {
	PlayerPermissionLevel map[string]int
	ServerPermissionLevel map[string]int
}

var Plugin = proxy.Plugin{
	Name: "TestPlugin",
	Init: func(ctx context.Context, p *proxy.Proxy) error {
		Client := http.Client{}
		PermissionLevel := Permissionstruct{
			PlayerPermissionLevel: map[string]int{
				"CaoskingYT32": 1,
			},
			ServerPermissionLevel: map[string]int{
				"private1": 1,
			},
		}

		models := models{
			controller: controllerModel{
				address: address{
					ip:   "127.0.0.1",
					port: 8080,
				},
				secret: "secret",
			},
			proxy: proxyModel{
				address: address{
					ip:   "127.0.0.1",
					port: 8090,
				},
				kind:   "default",
				secret: "secret2",
			},
		}

		log := logr.FromContextOrDiscard(ctx)
		log.Info("Hello from TestPlugin plugin!")

		registerProxy(models, Client)
		// Use ServerPostConnectEvent instead of onPostLogin for titles as they wont be displayed otherwise
		event.Subscribe(p.Event(), 0, onPostLogin)
		event.Subscribe(p.Event(), 0, func(e *proxy.ServerPostConnectEvent) {
			serverPostConnectEvent(PermissionLevel, e)
		})
		event.Subscribe(p.Event(), 1, func(e *proxy.PlayerChooseInitialServerEvent) {
			playerChooseInitialServerEvent(p, e)
		})
		event.Subscribe(p.Event(), 2, func(e *proxy.PreShutdownEvent) {
			preShutdownEvent(p, e, models, Client)
		})
		event.Subscribe(p.Event(), 3, func(e *proxy.ServerPreConnectEvent) {
			serverPreConnectEvent(PermissionLevel, e)
		})
		// event.Subscribe(p.Event(), 4, func(e *proxy.PlayerChatEvent) {
		//	playerChatEvent(PermissionLevel, p, e)
		// })

		return nil
	},
}

func serverPostConnectEvent(p Permissionstruct, e *proxy.ServerPostConnectEvent) {
	player := e.Player()
	welcomeText := Text(fmt.Sprintf("Hello &6&l%s&r!", player.Username()))
	time := time.Now()
	timeLayout := time.Format("Monday, 02 January 06 15:04:05 MST")
	subTitle := Text(fmt.Sprintf("&k1&r You are logging in on %s &k1&r", timeLayout))
	title.ShowTitle(player, &title.Options{
		Title:    welcomeText,
		Subtitle: subTitle,
	})
	if e.PreviousServer() == nil {
		player.SendMessage(Text("Welcome to the server!"))
		if p.PlayerPermissionLevel[player.Username()] > 0 {
			player.SendMessage(Text("You are a member of the staff!"))
			player.SendMessage(Text("There are no server issues at the moment"))
		}
	}

}

func playerChooseInitialServerEvent(p *proxy.Proxy, e *proxy.PlayerChooseInitialServerEvent) {
	servers := p.Servers()
	r := regexp.MustCompile(`^lobby.*\d$`)
	var lobbys []proxy.RegisteredServer
	if len(servers) == 0 {
		e.SetInitialServer(nil)
		return
	}
	for _, server := range servers {
		if r.MatchString(server.ServerInfo().Name()) {
			if server.Players().Len() == 0 {
				e.SetInitialServer(server)
				return
			}
			lobbys = append(lobbys, server)
		}
	}
	if len(lobbys) == 0 {
		e.SetInitialServer(nil)
		return
	}
	initialServer := lobbys[0]
	for _, server := range lobbys {
		if server.Players().Len() < initialServer.Players().Len() {
			initialServer = server
		}
	}
	e.SetInitialServer(initialServer)
	return
}

func preShutdownEvent(p *proxy.Proxy, e *proxy.PreShutdownEvent, controllerData models, Client http.Client) {
	globalBroadcast(p, "Server is shutting down...\nPrepare to be disconnected after the preparations are complete")
	shutdownTODOs(controllerData, Client)
	i := 5
	globalBroadcast(p, fmt.Sprintf("Shutdownpreparations done.\nThe server is shutting down in %d seconds.\nGoodbye!", i))
	for i > 0 {
		globalBroadcast(p, fmt.Sprintf("Server shutdown in %d seconds...", i))
		i--
		time.Sleep(time.Second)
	}
}

func shutdownTODOs(controllerData models, client http.Client) {
	unregisterProxy(controllerData, client)
}

func globalBroadcast(p *proxy.Proxy, message string) error {
	for _, player := range p.Players() {
		err := player.SendMessage(Text(message))
		if err != nil {
			return err
		}
	}
	return nil
}

func serverPreConnectEvent(p Permissionstruct, e *proxy.ServerPreConnectEvent) {
	if p.PlayerPermissionLevel[e.Player().Username()] < p.ServerPermissionLevel[e.Server().ServerInfo().Name()] {
		e.Deny()
		e.Player().SendMessage(Text("You do not have permission to connect to this server!"))
	}
}

func onPostLogin(e *proxy.PostLoginEvent) {
	header := &c.Text{
		Content: "Hello &6&l" + e.Player().Username() + "&r!\n",
		S: c.Style{
			Color:      color.White,
			Bold:       c.True,
			Italic:     c.True,
			Underlined: c.True,
		},
	}
	footer := &c.Text{
		Content: "Welcome to the server!\n",
		S: c.Style{
			Color:  color.White,
			Italic: c.True,
		},
	}
	_ = e.Player().TabList().SetHeaderFooter(header, footer)
}

func playerChatEvent(s Permissionstruct, p *proxy.Proxy, c *proxy.PlayerChatEvent) {
	message := c.Message()
	globalBroadcast(p, fmt.Sprintf("[%d]%s: %s", s.PlayerPermissionLevel[c.Player().Username()], c.Player().Username(), message))
	c.SetAllowed(false)
	c.SetMessage("")
}

func registerProxy(models2 models, client http.Client) {
	url := fmt.Sprintf("http://%s:%d/listproxies", models2.controller.address.ip, models2.controller.address.port)
	getProxies, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	getProxies.Header.Set("Authorization", models2.controller.secret)
	response, err := client.Do(getProxies)
	if err != nil {
		panic(err)
	}
	var proxies []struct {
		Name    string  `json:"name"`
		Address address `json:"address"`
		Kind    string  `json:"kind"`
	}

	responseJSON := json.NewDecoder(response.Body)
	err = responseJSON.Decode(&proxies)
	if err != nil {
		panic(err)
	}
	var sameKindProxies []proxyModel
	for _, proxy := range proxies {
		if proxy.Kind == models2.proxy.kind {
			sameKindProxies = append(sameKindProxies, proxyModel{
				name:    proxy.Name,
				address: proxy.Address,
				kind:    proxy.Kind,
			})
		}
	}
	id := 1
	reg := regexp.MustCompile(fmt.Sprintf(`^%s([0-9]+)`, models2.proxy.kind))
	if len(sameKindProxies) != 0 {
		for _, proxy := range sameKindProxies {
			submatch := reg.FindStringSubmatch(proxy.name)
			if len(submatch) != 2 {
				panic(fmt.Sprintf("Proxy name %s is not a valid proxy name", proxy.name))
			}

			proxyID, err := strconv.Atoi(submatch[1])
			if err != nil {
				panic(err)
			}
			if proxyID == id {
				id++
				continue
			}
		}
	}
	models2.proxy.name = fmt.Sprintf("%s%d", models2.proxy.kind, id)
	proxyData := struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Kind    string `json:"kind"`
		Secret  string `json:"secret"`
	}{
		Name:    models2.proxy.name,
		Address: models2.proxy.address.String(),
		Kind:    models2.proxy.kind,
		Secret:  models2.proxy.secret,
	}

	proxyJSON, err := json.Marshal(proxyData)
	if err != nil {
		panic(err)
	}
	url = fmt.Sprintf("http://%s:%d/addproxy", models2.controller.address.ip, models2.controller.address.port)
	registerProxyRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(proxyJSON))
	if err != nil {
		panic(err)
	}
	registerProxyRequest.Header.Set("Authorization", models2.controller.secret)
	registerProxyRequest.Header.Set("Content-Type", "application/json")
	response, err = client.Do(registerProxyRequest)
	if err != nil {
		panic(err)
	}
	if response.StatusCode != 200 {
		panic(fmt.Sprintf("Failed to register proxy: %s", response.Status))
	}

}

func unregisterProxy(models2 models, client http.Client) {
}
