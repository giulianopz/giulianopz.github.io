---
layout: post
title:  "Full-duplex HTTP Streaming in Go"
date:   2023-11-28
categories: full-duplex http-streaming go web-speech-api
permalink: /http-streaming-in-go
---

### Wait! Backstory, first...

Few weeks ago, a friend of mine asked me for help after she couldn't connect to a web service essential to her daily life.

She was using a speech-to-text service named WebCaptioner to generate captions on the fly for any audio content not provided wit closed captions: uni lectures, Yoga classes, random stuff on YouTube, etc. Being deaf, that's the only way to access common information every human being needs to live on this planet.

Unfortunately, its author, [Curt Grimes](https://curtgrimes.com), stopped running the service October 31, 2023. Browsing to the site (`https://webcaptioner.com`) you are now redirected to a [repository](https://github.com/curtgrimes/webcaptioner) on GitHub hosting its source code. In fact, Curt generously decided to donate the source code of such service to the community including its full commit history.

The webapp is a server-side rendered Nuxt/Vue.js app. Someone had already forked it removing major issues to have it running, so I've just commented out additonal features no longer needed and deployed it to a fly.io machine. Problem fixed for this girl who has now a web service running on the cloud for free, spinning up everytime she needs. But the problem remains for the rest of the WebCaptioner users who are unaware of my little hack, since this simple app had a quite significant audience according to its author:

> Maintain high availability with 10,000 weekly users and over 40,000 account registrations while balancing affordability with AWS Fargate and ElastiCache. 
>
> (quote from his [résumé](https://curtgrimes.com/resume))

I was surprised when I saw it even mentioned by [an Italian university](https://site.unibo.it/studenti-con-disabilita-e-dsa/it/per-studenti/presentazione-di-strumenti-automatici-per-la-sottotitolazione-e-o-dettatura-vocale) among recommended softwares that can be used by students suffering from hearing loss or learning disabilities. 

Clearly, there's a desperate need for this kind of web services who are so essential to these people that can be run by individuals like Curtis or private companies, they should be owned and run by governments: the accessibility to this kind of tools cannot be constrained by a paid subscription nor it can rely solely on the unstable willingness of FOSS developers. This topic is so crucial that it deserves its own post...

Anyway I will try to do my part in helping anyone who still needs WebCaptioner by maintaining my own [fork](https://github.com/giulianopz/webcaptioner), but I'm not a JS person, nor even expert in web hosting solutions that can scale worldwide without costing billions of dollars, so if you can, your help is more than welcome.

### Backstory, again... Web Speech API on Chrome

That said, even before it was shutdowned, I was curious to understand how that web service was performing speech-to-text (STT) transcriptions without asking money to its users. I thought it was using some expensive super-mega-wow OpenAI services... but incredibly it turned out to be a mere web teleprompter relying on a thing called [Web Speech API](https://wicg.github.io/speech-api/), a W3C-supported JavaScript API specification to "enable web developers to incorporate speech recognition and synthesis into their web pages". Interestingly, the implementation of the recognition API is entirely browser- or device-dependent:

> Generally, the default speech recognition system available on the device will be used for the speech recognition — most modern OSes have a speech recognition system for issuing voice commands. Think about Dictation on macOS, Siri on iOS, Cortana on Windows 10, Android Speech, etc.
>
> Note: On some browsers, such as Chrome, using Speech Recognition on a web page involves a server-based recognition engine. Your audio is sent to a web service for recognition processing, so it won't work offline.
>
> ([MDN Web Docs](https://developer.mozilla.org/en-US/docs/Web/API/Web_Speech_API/Using_the_Web_Speech_API))

Here is a [demo page](https://www.google.com/intl/en/chrome/demos/speech.html) to test Web Speech API on Chrome.

The final note in the above quote explains why WebCaptioner only worked on Google Chrome: the microphone input was directly streamed to a STT service of Google. And it took me just a few Google search queries to find out that this service was running for many years and some people had already discovered how to call it without Javascript.

[Mike Pultz](https://mikepultz.com/2011/03/accessing-google-speech-api-chrome-11/) was possibly the first one to discover it in 2011. Subsequently, [Travis Payton](http://blog.travispayton.com/wp-content/uploads/2014/03/Google-Speech-API.pdf) published a detailed report on the subject.

Wrapping up: the original implementation was limited to short audio samples, but the current one has virtually no limits, being capable of transcribing also quite long audio files. It just requires an API key. You could ask for a key reserved to developers on the Chromium project [site](https://www.chromium.org/developers/how-tos/api-keys/). But the API key you need is actually built into Google Chrome, as you can see by poking your nose into the [Chromium source code](https://source.chromium.org/chromium/chromium/src/+/main:content/browser/speech/speech_recognition_engine.cc). And it's also the same for every Chrome installation (i.e. `AIzaSyBOti4mM-6x9WDnZIjIeyEU21OpBXqWBgw`), you can prove it by running the following command:
```bash
strings /opt/google/chrome/chrome | grep AIzaSyBOti4mM-6x9WDnZIjIeyEU21OpBXqWBgw
```

I guess this is the same service my Google Pixel connects to when [Live Caption](https://www.androidauthority.com/live-caption-pixel-3224653/) is enabled, but I had no chance to verify it so far. 

Curb your enthusiasm, anyway: this is undocumented API that is not guaranteed to exist in the future. Also note, one of the authors of the Web Speech API spec [said]((https://lists.w3.org/Archives/Public/public-speech-api/2013Jul/0001.html)) that commercial apps can be based upon it.   

### Finally, Go

So, looking at the above-linked analyses on this STT service, in its second version (so called "full duplex"), I couldn't understand at first how you can send unbounded streams of chunked data with a "simple" REST endpoint, mantaining a persistent TCP connection to receive back interim results representing transcripts of audio recorded straight from microphone. 

Due to my gross ignorance, I thought HTTP was unsuitable for real-time streaming and that you have to use something like Websockets, WebRTC or even WebTransport instead.

Altough HTTP was designed as a single request-response exchange (cfr. [rfc7540#8.1](https://datatracker.ietf.org/doc/html/rfc7540#section-8.1)), an Internet-Draft (cfr. [draft-zhu-http-fullduplex](https://datatracker.ietf.org/doc/html/draft-zhu-http-fullduplex-08)) was submitted to IETF to clarify the requirements of full-duplex HTTP on top of the basic HTTP protocol semantics:
> Full-duplex HTTP follows the basic HTTP request-response semantics but also allows the server to send response body to the client at the same time when the client is transmitting request body to the server. 

The draft is expired but it was welcomed by the HTTP/2 specification (cfr. [rfc7540#8.1](https://datatracker.ietf.org/doc/html/rfc7540#section-8.1)):
> A server can send a complete response prior to the client sending an entire request if the response does not depend on any portion of the request that has not been sent and received.

Indeed, the Google Speech API is a full-duplex HTTP streaming API, meaning that request and response are transmitted between client and server simultaneously over the same TCP connection, i.e.:
- a POST request to upload the audio content in chunks
- a GET response to receives the results back.

So, I wrapped my head around, trying to understand how to to do it in Go.

Reading the stream was easy since this what `json.Decoder` is supposed to do as shown in the package [documentation](https://pkg.go.dev/encoding/json#example-Decoder).

I struggled a little bit for the upload part. Then I understood from this [coversation](
https://groups.google.com/g/golang-nuts/c/LTem1EEHopc) that the idiomatic way for it was to use a `io.Pipe`:
> [Pipe](https://pkg.go.dev/io#Pipe) creates a synchronous in-memory pipe. It can be used to connect code expecting an io.Reader with code expecting an io.Writer. 

That was enough to simply make the microphone input flow into the outgoing request body. 

The code snippet is an abridged version of the client I wrote to call the Google Speech API: 
```go
// create a channel
out := make(chan *transcription.Response)

// loop over channel until it is closed by the sender
go func() {
    for resp := range out {
        for _, result := range resp.Result {
            for _, alt := range result.Alternative {
                fmt.Printf("confidence=%f, transcript=%s\n", alt.Confidence, strings.TrimSpace(alt.Transcript))
            }
        }
    }
}()

// create a pipe
pr, pw := io.Pipe()
defer pr.Close()
defer pw.Close()

// read audio from stdin in 1kB chunks until end of stream or error
bs := make([]byte, 1024)
for {
    n, err := os.Stdin.Read(bs)
    if n > 0 {
        // write input chunks to the pipe
        _, err := pw.Write(bs)
        if err != nil {
            panic(err)
        }
    } else if err == io.EOF {
        break
    } else if err != nil {
        panic(err)
    }
}

// receive transcripts from downstream
go func() {
    req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

    rsp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer rsp.Body.Close()

	// consume JSON messages from the body until end of stream or error
	dec := json.NewDecoder(rsp.Body)
	for {
		speechRecogResp := &transcription.Response{}

		err = dec.Decode(speechRecogResp)
		if err == io.EOF {
			close(out)
			break
		}
		if err != nil {
			panic(err)
		}
        // send response to the channel
		out <- speechRecogResp
	}
}()

// send chunks of input to the upstream with the pipe
go func() {
    req, err := http.NewRequest(http.MethodPost, url, pr)
	if err != nil {
		panic(err)
	}

	rsp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer rsp.Body.Close()
}()

```

You can find the complete code on [Github]((https://github.com/giulianopz/go-gstt)).

http3

server-side


