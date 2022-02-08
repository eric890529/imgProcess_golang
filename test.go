package main
import (
	"flag"
	"fmt"
	"log"
	"github.com/valyala/fasthttp"
	"github.com/nfnt/resize"
	"image/png"
	"image/jpeg"
	"image"
	"strconv"
	"bytes"
	"os"
	"io/ioutil"
	"strings"
	"path/filepath"
)


var (
	/*addr     = flag.String("addr", ":8080", "TCP address to listen to")*/
	/*host     = flag.String("host", "test", "TCP address to listen to")*/
	compress = flag.Bool("compress", false, "Whether to enable transparent response compression")
)

func main() {


	h := requestHandler
	if *compress {
		h = fasthttp.CompressHandler(h)
	}
	if err := fasthttp.ListenAndServe(addr, h); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
	//doRequest("https://static.wikia.nocookie.net/beatbox/images/0/0a/Colaps_Grand_Beatbox_Battle_2019.jpg/revision/latest?cb=20200624015609")
}

var addr string
var host string
var defaultConfig image.Config
var defaultType string
var defaultImageMap map[string]*defaultImg
var imgSize int
var defaultImage image.Image

type defaultImg struct {
    defaultImage image.Image
    defaultConfig image.Config
	defaultType string
}

func init(){
    //defaulte img
    flag.Parse()
	fmt.Printf("args=%s, num=%d\n", flag.Args(), flag.NArg())
	addr = "localhost:" + flag.Args()[0]
	host = flag.Args()[1]

	files, err := ioutil.ReadDir("./default_img/")
    if err != nil {
        log.Fatal(err)
    }
	
	defaultImageMap = make(map[string]*defaultImg)
    for _, file := range files {
		defaultImage, defaultConfig, defaultType, imgSize, _ = getImageFromFilePath("./default_img/" + file.Name())
		fileName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		defaultImageMap[fileName] = &defaultImg{
			defaultImage : defaultImage,
			defaultConfig : defaultConfig,
			defaultType : defaultType,
		}
		log.Printf("%d", imgSize)
    }
}

func getImageFromFilePath(filePath string) ( image.Image, image.Config, string, int, error) {
    f, err := os.Open(filePath)
	imgBytes, _ := ioutil.ReadAll(f);
    if err != nil {
        return nil,image.Config{},"", err
    }
    img, imageType, err := image.Decode(bytes.NewReader(imgBytes))
	im, _, _ := image.DecodeConfig(bytes.NewReader(imgBytes))
	fmt.Printf("The file is %d bytes long\n", len(imgBytes))
	defer f.Close()
    return img, im, imageType , 123, err
}




func requestHandler(ctx *fasthttp.RequestCtx) {
	host = flag.Args()[1]
	host = host + string(ctx.Path())
	fmt.Printf("host : %s\n\n",host)

	var scale float64
	query := string(ctx.QueryArgs().Peek("q"))
	defaultQuery := string(ctx.QueryArgs().Peek("default"))
	method := string(ctx.QueryArgs().Peek("method"))
	max := string(ctx.QueryArgs().Peek("max"))
	maxNum, _:= strconv.Atoi(max)
	maxNum = maxNum*1000

	if query != ""{
		var num string = query
		scale, _ = strconv.ParseFloat(num, 64)
	}else{
		scale = 1
	}
	resp := doRequest(host)
	bodyBytes := resp.Body()

	fmt.Printf("comapre")
	fmt.Printf("%d %d\n",len(bodyBytes),maxNum)
	if method == "UB" && len(bodyBytes) > maxNum{
		divNum := len(bodyBytes) * maxNum
		scale = float64(divNum)
		fmt.Printf("UB")
		fmt.Printf("%f",scale)
	}

	var respHeader fasthttp.ResponseHeader
	var contentType string
	var img image.Image
	var im image.Config
	
	if resp.StatusCode() == 200{
		respHeader = resp.Header
		contentType = string(respHeader.ContentType())
		fmt.Printf( "Header %q\n", respHeader.ContentType())
		img, _, _ = image.Decode(bytes.NewReader(bodyBytes))
		im, _, _ = image.DecodeConfig(bytes.NewReader(bodyBytes))
		send_s3 := compressImg(img, im, scale, contentType)
		response(ctx, send_s3, contentType)
	}else if resp.StatusCode() != 200 && defaultQuery != ""{
		contentType = "image/" + defaultImageMap[defaultQuery].defaultType
		//log.Println("resp.StatusCode = ",resp.StatusCode())
		//log.Println("defaultQuery = ",defaultQuery)
		img = defaultImageMap[defaultQuery].defaultImage
		im = defaultImageMap[defaultQuery].defaultConfig
		send_s3 := compressImg(img, im, scale, contentType)
		response(ctx ,send_s3, contentType)
	}else{
		log.Println("resp.StatusCode = ",resp.StatusCode())
		log.Println("err")
		log.Println(fasthttp.StatusUnsupportedMediaType )
		ctx.Error("Unsupported path", fasthttp.StatusUnsupportedMediaType )
	}
	fmt.Printf("size %d %d\n\n", im.Width, im.Height)
}

func response(ctx *fasthttp.RequestCtx, send_s3 []byte, contentType string){

	ctx.SetContentType(contentType)

	ctx.SetBody([]byte(send_s3))

	ctx.Response.Header.Set("X-My-Header", "my-header-value")
	// Set cookies
	var c fasthttp.Cookie
	c.SetKey("cookie-name")
	c.SetValue("cookie-value")
	ctx.Response.Header.SetCookie(&c)
}

func compressImg(img image.Image, im image.Config, scale float64, contentType string)([]byte){
	fmt.Printf( "origin size %d %d\n\n",im.Height, im.Width)
	h :=  float64(im.Height)*scale
	w :=  float64(im.Width)*scale
	fmt.Printf( "reize %f %f\n\n",h, w)

	m := resize.Resize( 0, uint(h), img, resize.Lanczos3)
	buf := new(bytes.Buffer)

	switch {
    case contentType == "image/jpeg":
        fmt.Println("jpeg")
		jpeg.Encode(buf, m, nil)
    case contentType == "image/png":
        fmt.Println("png")
		png.Encode(buf, m)
    case contentType == "image/jpg":
        fmt.Println("jpg")
		jpeg.Encode(buf, m, nil)
	default: //default:當前面條件都沒有滿足時將會執行此處內包含的方法
	    jpeg.Encode(buf, m, nil)
    }

	send_s3 := buf.Bytes()
	return send_s3
}

func doRequest(url string)(*fasthttp.Response) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(url)

	resp := fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	client.Do(req, resp)
	//println(string(bodyBytes))
	// User-Agent: fasthttp
	// Body:
	return resp
}