package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	dc "github.com/samalba/dockerclient"
)

func main() {
	app := cli.NewApp()
	app.Name = "docker-du"
	app.Usage = "Docker disk usage"
	app.Version = "0.1.0"
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
	client.buildImageTree("")
}

// Docker Client init code
type dclient struct {
	client *dc.DockerClient
}

func initDockerClient() (*dclient, error) {
	client, err := dc.NewDockerClient("unix:///var/run/docker.sock", nil)
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
	id            string
	parent        string
	actualSize    int64
	totalSize     int64
	tag           []string
	refrenceCount int
}

type ImageInfoList struct {
	imageInfo ImageInfo
	child     []ImageInfoList
}

func printImageTree(imageTree []ImageInfoList, tab int) {
	for _, subTree := range imageTree {
		printSubTree(subTree, tab)
	}
}

func printSubTree(tree ImageInfoList, tab int) {
	//for i := 0; i < tab/2; i++ {
	//	fmt.Printf("_")
	//}
	fmt.Printf("_  %14s  %d MB, %d\n", tree.imageInfo.id[0:12], tree.imageInfo.actualSize/(1024*1024), len(tree.child))
	for _, image := range tree.child {
		for i := 0; i < tab; i++ {
			fmt.Printf("  ")
		}
		fmt.Printf("|_")
		printSubTree(image, tab+1)
	}
}

func (dcl *dclient) buildImageTree(image string) {

	imageList, err := dcl.client.ListImages(false)
	if err != nil {
		fmt.Println("Error while fetching image list")
		return
	}

	var imageTree []ImageInfoList

	for _, image := range imageList {
		//if image.Id != "05b0290b4b5e63acba5007ee016d5de2142eeacb1400a65bffdaf01c6c3652d1" {
		if image.Id != "d2a0ecffe6fa4ef3de9646a75cc629bbd9da7eead7f767cb810f9808d6b3ecb6" &&
		image.Id != "05b0290b4b5e63acba5007ee016d5de2142eeacb1400a65bffdaf01c6c3652d1"{
			continue
		}
		temp, _ := dcl.client.InspectImage(image.Id)
		tree := dcl.buildImageTreeDetails(temp, imageTree, image.RepoTags[0])
		imageTree = tree

	}
	fmt.Println("-------------------- Final Tree ---------------")
	//fmt.Println(imageTree)
	printImageTree(imageTree, 0)
}

func (dcl *dclient) buildImageTreeDetails(image *dc.ImageInfo, imageTree []ImageInfoList, tag string) []ImageInfoList {
	s := NewStack()
	currentImage := image
	for currentImage != nil {
		s.Push(currentImage)
		if currentImage.Parent != "" {
			currentImage, _ = dcl.client.InspectImage(currentImage.Parent)
		} else {
			currentImage = nil
		}
	}

	node := s.Pop()
	if node == nil {
		panic(fmt.Errorf("Error, Stack cannot be empty"))
	}
	foundNode := checkRootExist(node.(*dc.ImageInfo), imageTree)
	if foundNode < 0 {
		fmt.Println("--- Node not found in Tree ---")
		imageTree = addNodeToImageTree(imageTree, node.(*dc.ImageInfo), s)
	} else {
		fmt.Println("--- Node found in Tree ---")
		imageTree = addToBranchNode(foundNode, s, imageTree)
	}
	return imageTree
}

func addNodeToImageTree(imageTree []ImageInfoList, image *dc.ImageInfo, s *Stack) []ImageInfoList {

	node := imageToNode(image)
	rootImage := ImageInfoList{node, nil}
	childList := addNodesAsChild(rootImage, s)
	rootImage.child = append(rootImage.child, childList)
	imageTree = append(imageTree, rootImage)

	//	fmt.Println(len(imageTree))

	return imageTree
}

func addNodesAsChild(image ImageInfoList, s *Stack) ImageInfoList {

	tempParent := s.Pop()
	if tempParent != nil {
		temp := tempParent.(*dc.ImageInfo)
	//	fmt.Println(temp.Id)
		node := ImageInfoList{imageToNode(temp), nil}
		node = addNodesAsChild(node, s)
		image.child = append(image.child, node)
	}

	return image
}

func addNodesAsChildQueue(image ImageInfoList, s *Stack) ImageInfoList {

	tempParent := s.Dequeue()
	if tempParent != nil {
		temp := tempParent.(*dc.ImageInfo)
	//	fmt.Println(temp.Id)
		node := ImageInfoList{imageToNode(temp), nil}
		node = addNodesAsChildQueue(node, s)
		image.child = append(image.child, node)
	}

	return image
}

func checkRootExist(node *dc.ImageInfo, imageTree []ImageInfoList) int {

	for i, image := range imageTree {
		if image.imageInfo.id == node.Id {
			return i
		}
	}
	return -1
}

func addToBranchNode(nodeIdx int, s *Stack, imageTree []ImageInfoList) []ImageInfoList {

	//Till depth of tree
	//	branchTree := imageTree
	//	for tree := range branchTree[nodeIdx:] {
	for i, tempNode := range imageTree[nodeIdx:] {
		stackNode := s.Peek().(*dc.ImageInfo)

		if tempNode.imageInfo.id != stackNode.Id {
			// Access to prev node in imageTree
			// Add node a +1 child of that node.
			prevNode := imageTree[i]
			// Before adding, pop from stack, and create a list and add them as list.
			queue := NewStack()

		
			node := addNodesAsChildQueue(prevNode, s)
			prevNode.child = append(prevNode.child, node)
			imageTree[i] = prevNode

			break
		}
		if s.Peek() == nil {
			break
		}
		stackNode = s.Pop().(*dc.ImageInfo)
	}
	fmt.Println("== Printing intertnally == ")
	printImageTree(imageTree, 0)
	return imageTree

}

func imageToNode(image *dc.ImageInfo) ImageInfo {

	var node ImageInfo
	if image == nil {
		return ImageInfo{}
	}
	node.parent = image.Parent
	node.actualSize = image.Size
	node.totalSize = image.VirtualSize
	node.id = image.Id
	return node
}
