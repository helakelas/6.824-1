package mapreduce

import (
	"bufio"
	"bytes"
	"encoding/json"
	"hash/fnv"
	"log"
	"github.com/Alluxio/alluxio-go/option"
	"fmt"
)

func CheckError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}


// doMap manages one map task: it reads one of the input files
// (inFile), calls the user-defined map function (mapF) for that file's
// contents, and partitions the output into nReduce intermediate files.
func doMap(
	jobName string, // the name of the MapReduce job
	mapTaskNumber int, // which map task this is
	inFile string,
	nReduce int, // the number of reduce task that will be run ("R" in the paper)
	//mapTaskNumber和nReduce不一样，第一个变量指示的是map任务，第二个任务指示的reduce任务的个数!!
	mapF func(file string, contents string) []KeyValue,
) {
	//在sequentialsingle中得的infile是makeinput()制造出来的，nreduce=1

	debug("doMap: %s\n", jobName)
	//将map中的内容读入到一个用空格分隔的string中

	/*fileHandle, err := os.Open(inFile)
	if err != nil {
		log.Fatal("fatal error in opening infile", err)
	}*/
	fs := SetUpClient("10.2.152.24")
	readId, err := fs.OpenFile("/test/"+inFile, &option.OpenFile{})
	if err != nil {
		log.Fatal("doMap: read file from alluxio: ", err)
	}
	fileHandle, err:= fs.Read(readId)
	if err != nil {
		log.Fatal(err)
	}
	defer fs.Close(readId)
	var buffer bytes.Buffer
	fileScanner := bufio.NewScanner(fileHandle)
	for fileScanner.Scan() {
		buffer.WriteString(fileScanner.Text() + string(' '))
	}
	fileContents := buffer.String()
	sliceFileContents := mapF(inFile, fileContents) //slicefilecontens is the resulting key/value pairs
	//fmt.Println(sliceFileContents)
	whichFile := make(map[int]*json.Encoder)
	//create the mrtmp.xxx-i-j file and save the reduceindex2file map
	createId :=make([]int, nReduce)
	ioBuff := make([]bytes.Buffer, nReduce)
	for j := 0; j < nReduce; j++ {
		mrTmpFileName := reduceName(jobName, mapTaskNumber, j)
		/*tmpfileHandle, err := os.Create(mrTmpFileName)
		if err != nil {
			log.Fatal("create file error")
		}
		defer tmpfileHandle.Close()
		whichFile[j] = json.NewEncoder(tmpfileHandle)*/
		fmt.Println("creating mrtmp.wcseq-",mapTaskNumber,"-",j)
		createId[j] ,err = fs.CreateFile("/test/"+mrTmpFileName, &option.CreateFile{})
		if err != nil {
			log.Fatal(err)
		}
		whichFile[j] = json.NewEncoder(&ioBuff[j])
	}
	for _, kv := range sliceFileContents { //determine which reduce task intermediate file this key hash to
		//要进行运算的时候采用结构体，要写入文件使用json格式，中间的解码与编码采用json的marshall, unmarshall, decode, encode
		reduceTaskNumber := ihash(kv.Key) % nReduce
		err := whichFile[reduceTaskNumber].Encode(&kv)
		CheckError(err)
	}
	for j:= 0;j < nReduce;j++ {
		_, err= fs.Write(createId[j], &ioBuff[j])
		if err != nil {
			log.Fatal(err)
		}
		fs.Close(createId[j])
	}
	//
	// You will need to write this function.
	//
	// The intermediate output of a map task is stored as multiple
	// files, one per destination reduce task. The file name includes
	// both the map task number and the reduce task number. Use the
	// filename generated by reduceName(jobName, mapTaskNumber, r) as
	// the intermediate file for reduce task r. Call ihash() (see below)
	// on each key, mod nReduce, to pick r for a key/value pair.
	//
	// mapF() is the map function provided by the application. The first
	// argument should be the input file name, though the map function
	// typically ignores it. The second argument should be the entire
	// input file contents. mapF() returns a slice containing the
	// key/value pairs for reduce; see common.go for the definition of
	// KeyValue.
	//
	// Look at Go's ioutil and os packages for functions to read
	// and write files.
	//
	// Coming up with a scheme for how to format the key/value pairs on
	// disk can be tricky, especially when taking into account that both
	// keys and values could contain newlines, quotes, and any other
	// character you can think of.
	//
	// One format often used for serializing data to a byte stream that the
	// other end can correctly reconstruct is JSON. You are not required to
	// use JSON, but as the output of the reduce tasks *must* be JSON,
	// familiarizing yourself with it here may prove useful. You can write
	// out a data structure as a JSON string to a file using the commented
	// code below. The corresponding decoding functions can be found in
	// common_reduce.go.
	//
	//   enc := json.NewEncoder(file)
	//   for _, kv := ... {
	//     err := enc.Encode(&kv)
	//
	// Remember to close the file after you have written all the values!
	//
}

func ihash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32() & 0x7fffffff)
}
