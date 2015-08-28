package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	dc "github.com/samalba/dockerclient"
	"encoding/json"
	"bytes"
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
	Id            string
	Parent        string
	ActualSize    int64
	TotalSize     int64
	Tag           []string
	RefrenceCount int
}

type ImageInfoList struct {
	Image	 ImageInfo	`json:"image-info"`
	Child     []ImageInfoList `json:"image-list"`
}

func printImageTree(imageTree []ImageInfoList, tab int) {
	for _, subTree := range imageTree {
		//		fmt.Println(subTree)
		//fmt.Println("---------------------")
		printSubTree(subTree, tab)
	}
}

func printSubTree(tree ImageInfoList, tab int) {
	//for i := 0; i < tab/2; i++ {
	//	fmt.Printf("_")
	//}
	fmt.Printf("_  %14s  %d MB, %d\n", tree.Image.Id[0:12], tree.Image.ActualSize/(1024*1024), len(tree.Child))
	for _, image := range tree.Child {
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
		if image.Id != "ca0c37cd6ae78c9854913d81d366ec10aa40b1e17e3bfa9b8da87b1cdf45755f" &&
			image.Id != "05b0290b4b5e63acba5007ee016d5de2142eeacb1400a65bffdaf01c6c3652d1" {
			continue
		}
		temp, _ := dcl.client.InspectImage(image.Id)
		tree := dcl.buildImageTreeDetails(temp, imageTree, image.RepoTags[0])
		imageTree = tree


	}
	fmt.Println("-------------------- Final Tree ---------------")
	//fmt.Println(imageTree)
	//dumpJSON(imageTree)
	//printImageTree(imageTree, 0)
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
		printStack(*s)
		imageTree = addNodeToImageTree(imageTree, node.(*dc.ImageInfo), s)
	} else {
		fmt.Println("--- Node found in Tree ---")
		printStack(*s)
		imageTree = addToBranchNode(foundNode, s, imageTree)
	}
	return imageTree
}

func addNodeToImageTree(imageTree []ImageInfoList, image *dc.ImageInfo, s *Stack) []ImageInfoList {

	node := imageToNode(image)
	rootImage := ImageInfoList{node, nil}
	childList := addNodesAsChild(rootImage, s)
	rootImage.Child = append(rootImage.Child, childList)
	imageTree = append(imageTree, rootImage)

	//	fmt.Println(len(imageTree))

	return imageTree
}

func addNodesAsChild(image ImageInfoList, s *Stack) ImageInfoList {

	tempParent := s.Pop()
	if tempParent != nil {
		temp := tempParent.(*dc.ImageInfo)
	//	fmt.Println(temp.Id[:12])
		node := ImageInfoList{imageToNode(temp), nil}
		node = addNodesAsChild(node, s)
		image.Child = append(image.Child, node)
	}

	return image
}


func checkRootExist(node *dc.ImageInfo, imageTree []ImageInfoList) int {

	for i, image := range imageTree {
		if image.Image.Id == node.Id {
			return i
		}
	}
	return -1
}

func addToBranchNode(nodeIdx int, s *Stack, imageTree []ImageInfoList) []ImageInfoList {

	for i, tempNode := range imageTree[nodeIdx:] {
	//	printSubTree(tempNode, 0)
	//	fmt.Println("Size of Stack", s.Size())
	//	printStack(*s)
		tempNode, found := findAndPushBranchNode(tempNode, *s)
		if found {
			imageTree[i] = tempNode
			break
		}
	}
	return imageTree
}

func findAndPushBranchNode(subTree ImageInfoList, s Stack) (ImageInfoList, bool) {
	// TODO:
	// search recevesivly (dfs) in subtree, 
	// The node, at which stack and tree differ, add the stack nodes as sub tree


	//dumpJSON(subTree.Child, " Before ")
	found := false
	elem := s.Peek()
	if elem == nil {
		fmt.Println("===== Stack Empty ==== ")
		return subTree, false
	}
	node := elem.(*dc.ImageInfo)

	for _, tempNode := range subTree.Child {
		fmt.Printf("Cmp: %s <---> %s \n", tempNode.Image.Id[:12],  node.Id[:12])
		if tempNode.Image.Id == node.Id {
			s.Pop()
			fmt.Println("Popped:  ", node.Id[:12])
			found = true
			//dumpJSON(tempNode.Child,"found loop")
		}
		{
			tempNode, found = findAndPushBranchNode(tempNode, s)
			if found {
				fmt.Println("breaking...")
				break
			}
		}
	}

	// The common path ends here, Add as child to this subTree.
	return subTree, found
}

func addSubTree(subTree ImageInfoList, s Stack) ImageInfoList {

	newTree := addNodesAsChild(subTree, &s)
	dumpJSON(newTree.Child," newTree ")
	subTree.Child = append(subTree.Child, newTree)
	dumpJSON(subTree.Child," After ")
	return subTree
}

func printStack(s Stack) {
	fmt.Println("===== Stack ====")
	for elem := s.Pop(); elem != nil; elem = s.Pop() {
		node := elem.(*dc.ImageInfo)
		fmt.Println(node.Id)
	}
	fmt.Println("===== End ====")
}

/*	
	
	
	if tempNode.Image.Id != stackNode.Id {
			// Access to prev node in imageTree
			// Add node a +1 child of that node.
			prevNode := imageTree[i]
			// Before adding, pop from stack, and create a list and add them as list.
			//	queue := NewStack()

			//	for n := s.P(); n != nil {
			//		queue.Push(n)
			//	}

			fmt.Println("== Printing Append == ")
			node := addNodesAsChild(prevNode, s)
			prevNode.Child = append(prevNode.Child, node)
			imageTree[i] = prevNode

			fmt.Println("== Printing intertnally == ")
			//		printSubTree(prevNode, 0)
			fmt.Println("== Printing intertnally == ")
			break
		}
		if s.Peek() == nil {
			break
		}
		stackNode = s.Pop().(*dc.ImageInfo)
	}
	return imageTree

}
*/

func imageToNode(image *dc.ImageInfo) ImageInfo {

	var node ImageInfo
	if image == nil {
		return ImageInfo{}
	}
	node.Parent = image.Parent
	node.ActualSize = image.Size
	node.TotalSize = image.VirtualSize
	node.Id = image.Id
	return node
}

func dumpJSON(config []ImageInfoList, tag string) {
	b, err := json.Marshal(config)
	if err != nil {
		fmt.Println(err)
		return
	}
	var out bytes.Buffer
	fmt.Println("================= Start", tag, "===================")
	json.Indent(&out, b, "", "\t")
	out.WriteTo(os.Stdout)
	fmt.Println("================= End", tag, "===================")
	//	fmt.Println(out)
}
