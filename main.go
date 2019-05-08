package main

import (
	"context"
	"log"
	"os"
	"path"
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

func pickReReplace(dict map[string]string, re string, repl string) map[string]string {
	res := map[string]string{}
	r, err := regexp.Compile(re)
	if err != nil {
		return res
	}
	for k, v := range dict {
		if r.MatchString(k) {
			res[r.ReplaceAllString(k, repl)] = v
		}
	}
	return res
}

func pickRe(dict map[string]string, re string) map[string]string {
	return pickReReplace(dict, re, "$0")
}

type TemplateClient struct {
	cli *client.Client
}

func (tpl TemplateClient) funcs() template.FuncMap {

	cli := tpl.cli
	ctx := context.Background()

	return template.FuncMap{
		"pickRe":        pickRe,
		"pickReReplace": pickReReplace,

		"containers": func() []types.Container {
			containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
			if err != nil {
				log.Print(err)
				return []types.Container{}
			}
			return containers
		},

		"services": func() []swarm.Service {
			services, err := cli.ServiceList(ctx, types.ServiceListOptions{})
			if err != nil {
				log.Print(err)
				return []swarm.Service{}
			}
			return services
		},

		"tasks": func() []swarm.Task {
			tasks, err := cli.TaskList(ctx, types.TaskListOptions{})
			if err != nil {
				log.Print(err)
				return []swarm.Task{}
			}
			return tasks
		},

		"networks": func() []types.NetworkResource {
			resp, err := cli.NetworkList(ctx, types.NetworkListOptions{})
			if err != nil {
				log.Print(err)
				return []types.NetworkResource{}
			}
			return resp
		},

		"containerInspect": func(id string) types.ContainerJSON {
			resp, err := cli.ContainerInspect(ctx, id)
			if err != nil {
				log.Print(err)
				return types.ContainerJSON{}
			}
			return resp
		},

		"serviceInspect": func(id string) swarm.Service {
			resp, _, err := cli.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
			if err != nil {
				log.Print(err)
				return swarm.Service{}
			}
			return resp
		},

		"networkInspect": func(id string) types.NetworkResource {
			resp, err := cli.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
			if err != nil {
				log.Print(err)
				return types.NetworkResource{}
			}
			return resp
		},

		"taskInspect": func(id string) swarm.Task {
			resp, _, err := cli.TaskInspectWithRaw(ctx, id)
			if err != nil {
				log.Print(err)
				return swarm.Task{}
			}
			return resp
		},
	}

}

func main() {

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatal(err)
	}

	var f *os.File
	f = os.Stdout

	outFile := "stdout"
	if varOutFile, ok := os.LookupEnv("OUT_FILE"); ok {
		if varOutFile != "-" {
			f, err = os.Create(varOutFile)
			if err != nil {
				log.Fatal(err)
			}
			outFile = varOutFile
		}
	}

	tplCli := TemplateClient{cli}

	tplFile := os.Getenv("TPL_FILE")
	tplName := path.Base(tplFile)

	tpl := template.New(tplName).Funcs(sprig.TxtFuncMap()).Funcs(tplCli.funcs())
	tpl, err = tpl.ParseFiles(tplFile)
	if err != nil {
		log.Fatal(err)
	}
	if err := tpl.Execute(f, events.Message{}); err != nil {
		log.Fatal(err)
	}
	log.Printf("Generated on startup in %s", outFile)

	filters := filters.NewArgs()
	filters.Add("event", "create")
	filters.Add("event", "remove")
	filters.Add("event", "destroy")
	filters.Add("event", "update")
	filters.Add("type", events.ContainerEventType)
	filters.Add("type", events.ServiceEventType)
	filters.Add("type", events.NetworkEventType)

	events, errs := cli.Events(context.Background(), types.EventsOptions{
		Filters: filters,
	})

	go func() {
		for {
			select {
			case message := <-events:
				f.Truncate(0)
				f.Seek(0, 0)
				if err := tpl.Execute(f, message); err != nil {
					log.Fatal(err)
				}
				log.Printf("Generated on %s-%s in %s", message.Type, message.Action, outFile)
			}
		}
	}()

	<-errs

}