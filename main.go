package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/samalba/dockerclient"
)

func main() {
	app := cli.NewApp()
	app.Name = "docker-du"
	app.Usage = "Docker disk usage"
	app.Version = "0.0"
	app.Commands = []cli.Command{
		{
			Name:  "images",
			Usage: "Disk usgae by images only",
			Action: func(c *cli.Context) {
				imageDiskUsage(c)
			},
		},
	}
	app.Run(os.Args)
}

func imageDiskUsage(c *cli.Context) {
	client, _ := initDockerClient()
	client.getImageDiskUsage("---")
}

// Docker Client init code
type dclient struct {
	client *dockerclient.DockerClient
}

func initDockerClient() (*dclient, error) {
	client, err := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	dcl := &dclient{client: client}
	return dcl, nil
}

type ImageInfo struct {
	parent        string
	actualSize    int64
	totalSize     int64
	tag	[]string
	refrenceCount int
}

type ImageInfoNode struct {
	imageInfo ImageInfo
	parent    *ImageInfoNode
}

func (dcl *dclient) getImageDiskUsage(image string) {
	fmt.Println(image)
	imageList, err := dcl.client.ListImages(true)
	if err != nil {
		fmt.Println("Error while fetching image list")
		return
	}

	imageMap := make(map[string]ImageInfoNode)

	for _, image := range imageList {
		fmt.Println(image.Id, image.Size/1048576, "MB", image.VirtualSize/1048576)
		imageMap[image.Id] = dcl.getImageTree(image, imageMap)
	}

	fmt.Println("\n", imageMap)

}

func (dcl *dclient) getImageTree(image *dockerclient.Image, imageMap map[string]ImageInfoNode) *ImageInfoNode {

	if image.ParentId == "" {
		return ImageInfoNode{parent:nil}
	}

	if image.ParentId != "" {

		//Find the Node in map.

		//If not found in map, add details.
		var parentNode ImageInfoNode
		parentNode.imageInfo.parent = image.ParentId
		parentNode.imageInfo.actualSize = image.Size
		parentNode.imageInfo.totalSize = image.VirtualSize
		parentNode.imageInfo.tag = image.RepoTags
		//TODO: fix later
		parentNode.imageInfo.refrenceCount = 0

		imageMap[image.Id] = parentNode
		parentImage,_ := dcl.client.InspectImage(image.Id)
		parentNode.parent = dcl.getImageTree(parentImage, imageMap)
		return &parentNode
	}
		return ImageInfoNode{parent:nil}
}
