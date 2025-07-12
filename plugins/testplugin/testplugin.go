package testplugin

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/robinbraemer/event"
	"go.minekube.com/common/minecraft/color"
	c "go.minekube.com/common/minecraft/component"
	"regexp"
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
		PermissionLevel := Permissionstruct{
			PlayerPermissionLevel: map[string]int{
				"CaoskingYT32": 1,
			},
			ServerPermissionLevel: map[string]int{
				"private1": 1,
			},
		}

		log := logr.FromContextOrDiscard(ctx)
		log.Info("Hello from TestPlugin plugin!")

		// Use ServerPostConnectEvent instead of onPostLogin for titles as they wont be displayed otherwise
		event.Subscribe(p.Event(), 0, onPostLogin)
		event.Subscribe(p.Event(), 0, func(e *proxy.ServerPostConnectEvent) {
			serverPostConnectEvent(PermissionLevel, e)
		})
		event.Subscribe(p.Event(), 1, func(e *proxy.PlayerChooseInitialServerEvent) {
			playerChooseInitialServerEvent(p, e)
		})
		event.Subscribe(p.Event(), 2, func(e *proxy.PreShutdownEvent) {
			preShutdownEvent(p, e)
		})
		event.Subscribe(p.Event(), 3, func(e *proxy.ServerPreConnectEvent) {
			serverPreConnectEvent(PermissionLevel, e)
		})
		event.Subscribe(p.Event(), 4, func(e *proxy.PlayerChatEvent) {
			playerChatEvent(PermissionLevel, p, e)
		})

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
	for _, server := range servers {
		if r.MatchString(server.ServerInfo().Name()) {
			if server.Players().Len() == 0 {
				e.SetInitialServer(server)
				return
			}
			lobbys = append(lobbys, server)
		}
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

func preShutdownEvent(p *proxy.Proxy, e *proxy.PreShutdownEvent) {
	globalBroadcast(p, "Server is shutting down...\nPrepare to be disconnected after the preparations are complete")
	shutdownTODOs()
	i := 5
	globalBroadcast(p, fmt.Sprintf("Shutdownpreparations done.\nThe server is shutting down in %d seconds.\nGoodbye!", i))
	for i > 0 {
		globalBroadcast(p, fmt.Sprintf("Server shutdown in %d seconds...", i))
		i--
		time.Sleep(time.Second)
	}
}

func shutdownTODOs() {
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
	// c.SetAllowed(false)
	// c.SetMessage("")
}
