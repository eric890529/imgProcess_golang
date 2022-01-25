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
var defaultImage image.Image
var defaultConfig image.Config
var defaultType string
func init(){
//defaulte img
    flag.Parse()
	fmt.Printf("args=%s, num=%d\n", flag.Args(), flag.NArg())
	addr = "localhost:" + flag.Args()[0]
	host = flag.Args()[1]
	defaultImage, defaultConfig, defaultType, _ = getImageFromFilePath("./default_img/test.jpg")
	fmt.Printf(defaultType)
}

func getImageFromFilePath(filePath string) (image.Image, image.Config, string, error) {
    f, err := os.Open(filePath)
	imgBytes, _ := ioutil.ReadAll(f);
    if err != nil {
        return nil,image.Config{},"", err
    }
    defer f.Close()
    img, imageType, err := image.Decode(bytes.NewReader(imgBytes))
	im, _, _ := image.DecodeConfig(bytes.NewReader(imgBytes))
	log.Println(im.Width,im.Height)
	log.Println(imageType)
    return img, im, imageType, err
}


func requestHandler(ctx *fasthttp.RequestCtx) {
	host = flag.Args()[1]
	host = host + string(ctx.Path())
	fmt.Printf("host : %s\n\n",host)

	var scale float64
	query := string(ctx.QueryArgs().Peek("q"))
	defaultQuery := string(ctx.QueryArgs().Peek("default"))
	fmt.Printf( "defaultQuery is %q\n\n", defaultQuery)

	if query != ""{
		var num string = query
		scale, _ = strconv.ParseFloat(num, 64) 
	}else{
		scale = 1
	}
	
	fmt.Printf( "scale %f\n\n", scale)

	resp := doRequest(host)
	bodyBytes := resp.Body()

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
		contentType = "image/" + defaultType
		log.Println("resp.StatusCode = ",resp.StatusCode())
		log.Println("defaultQuery = ",defaultQuery)
		img = defaultImage
		im = defaultConfig
		send_s3 := compressImg(img, im, scale, contentType)
		response(ctx ,send_s3, contentType)
	}else{
		log.Println("resp.StatusCode = ",resp.StatusCode())
		log.Println("err")
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