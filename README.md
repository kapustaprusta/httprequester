# httprequester
The tool for converting a site response to an MD5 hash

## Downloading
For downloading the tool you should launch the next command
```
git clone git@github.com:kapustaprusta/httprequester.git
```

## Building
For building the tool you should launch the next commands
```
cd httprequester
make
```

## Testing
For running of all tests you should launch the next command
```
make test
```

## Installing
For installing the tool you should launch the next command
```
make install
````

## How it works
You should specify a list of urls to visit separated by whitespace:
```
httprequester url1 url2 url3 ...
```

For example:
```
httprequester http://www.adjust.com google.com facebook.com yahoo.com
```

Output:
```
http://www.adjust.com e2cf5d7a0850710f47b640f0412c2654
http://google.com 3b87e52025c45a3220c26050ceb5e315
http://facebook.com 486bf444e1770ca1f3bc4242228f99a5
http://yahoo.com f898480da8aa9b5214395da8ee819933
```

If you want specify number of parallel workers you should use parallel flag (`defaut 10`)
```
./httprequester --parallel 3 url1 url2 url3 ...
```

