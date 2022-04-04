# Captcha Server
A middleware server that warps https://github.com/steambap/captcha .



![](0.png)


### config
The configuration file is not required.
Once you captcha-serv is running, it will check the config file every 10 seconds, you won't need restart service.

Here is the example of a config file:
```yaml
#server
#listen addr
addr: 127.0.0.1:9001
#forward request to backend
backend: 127.0.0.1:8899
#forward host from header
forward_host: false


#client
#Validity period of the captcha
expire: 30
#request queue count
max: 5e4
#mini block time (second)
#It will increases as try times increases
wait: 300
#try times allow
block: 6

#captcha
#random char from char_preset
char_preset: 0123456789abcdef
#text front 
font: ./font.ttf
#image width
width: 150
height: 50
#text length
length: 5
noise: 1
curve: 4
background: 0xffffff
dpi: 60
#text colors
colors:
  - 0x06d6a0
  - 0x118ab2
  - 0x073b4c
  - 0xef476f
  - 0x708d81
  - 0x723d46
  - 0x3d405b
  - 0xe07a5f
```
             

### usage   
```bash
captcha-serv -cfg /xxx/cfg.yml
```

### caddy example
demo https://captcha.ivi.cx/

Caddyfile:
```
captcha.ivi.cx {
    @captcha {
          path /captcha
          method GET
          header !x-captcha-code
    }
    @hi {
        path /hi
        header x-captcha-code *
        header x-captcha-key *
        method POST
    }

    file_server / {
      root /root/www
      index index.html
    }

    reverse_proxy @captcha 127.0.0.1:9001
    reverse_proxy @hi 127.0.0.1:9001
}

http://127.0.0.1:9002 {
 respond "{\"data\":\"Hello world!\"}" 200
}
```