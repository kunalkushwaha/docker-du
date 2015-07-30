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

/*
Here the image-info need to be parsed and stored in a list of tree structure.
- All root of trees will be stored in list.
- Root should be root of images, and leaf will the images visible in `docker images` command.
*/

type ImageInfo struct {
	parent        string
	actualSize    int64
	totalSize     int64
	tag           []string
	refrenceCount int
}

type ImageInfoNode struct {
	imageInfo ImageInfo
	parent    *ImageInfoNode
}

func (dcl *dclient) getImageDiskUsage(image string) {
	fmt.Println(image)
	imageList, err := dcl.client.ListImages(false)
	if err != nil {
		fmt.Println("Error while fetching image list")
		return
	}

	imageMap := make(map[string]*ImageInfoNode)
	count := 0
	for _, image := range imageList {
		temp, _ := dcl.client.InspectImage(image.Id)
		imageMap[image.Id] = dcl.getImageTree(temp, imageMap)
		imageMap[image.Id].imageInfo.tag = image.RepoTags
	}
	count = 0
	for k, _ := range imageMap {
		dumpImageTree(imageMap[k], 0)
		fmt.Println("---------------------------")
		count++
	}
	fmt.Println("Total leaf Images ", count)

}

func dumpImageTree(image *ImageInfoNode, treeDepth int) bool {
	if image.parent == nil {
		return false
	}
	//for i := 0; i < treeDepth; i++ {
	if treeDepth > 0 {
		fmt.Printf("--")
	}
	//	refc := image.imageInfo.refrenceCount

	fmt.Printf(" %d  %12s %6d MB %6d MB %s\n", image.imageInfo.refrenceCount, image.imageInfo.parent, image.imageInfo.actualSize/(1024*1024), image.imageInfo.totalSize/(1024*1024), image.imageInfo.tag)

	return dumpImageTree(image.parent, treeDepth+1)

}

func (dcl *dclient) getImageTree(image *dockerclient.ImageInfo, imageMap map[string]*ImageInfoNode) *ImageInfoNode {

	if image.Parent == "" {
		return &ImageInfoNode{parent: nil}
	}

	if image.Parent != "" {

		//	fmt.Println(image.Id)
		//If not found in map, add details.
		parentNode := new(ImageInfoNode)
		parentNode.imageInfo.parent = image.Parent[0:12]
		parentNode.imageInfo.actualSize = image.Size
		parentNode.imageInfo.totalSize = image.VirtualSize
		//		parentNode.imageInfo.tag = image.RepoTags
		//TODO: fix later
		parentNode.imageInfo.refrenceCount = 0

		//Find the Node in map.
		if imageMap[image.Id[0:12]] != nil {
			temp := imageMap[image.Id]
			temp.imageInfo.refrenceCount++
			return temp
		}

		foundNode, res := findImageTree(image.Id[0:12], imageMap)
		if res {
			return foundNode
		}
		parentImage, _ := dcl.client.InspectImage(image.Parent)
		parentNode.parent = dcl.getImageTree(parentImage, imageMap)
		//fmt.Println("Return..", parentNode.imageInfo.parent)
		return parentNode
	}
	return &ImageInfoNode{parent: nil}
}

func findImageTree(imageId string, imageMap map[string]*ImageInfoNode) (*ImageInfoNode, bool) {

	//Loop in map,
	// and traverse each list of map item, using ImageWalk.
	for _, v := range imageMap {
		image, res := ImageWalk(imageId, v)
		if res {
			return image, res
		}
	}
	return &ImageInfoNode{parent: nil}, false

}

func ImageWalk(imageId string, image *ImageInfoNode) (*ImageInfoNode, bool) {
	if image.parent == nil {
		return &ImageInfoNode{parent: nil}, false
	}
	if image.imageInfo.parent == imageId {
		image.imageInfo.refrenceCount++
		return image.parent, true
	}
	return ImageWalk(imageId, image.parent)

}
