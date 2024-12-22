# annoyingtts
Frontend for TikToks Text to Speech/Sing Api

---
Today i woke up and chose violence - Sneed

---
## Usage
Messages are in stderr and the output goes to stdout.  

Usage of ./biehdc.priv.tiktoktts:  
*   -in string  
    *   pass text in instead of reading from stdin  
*   -verbose
    *   helps you figure out the chunking when stuff sounds weird and cut off (currently 200 chars)
*   -voice string  
    *   the voice to use (blank means random)  
*   -voices  
    *   print out the available voices and exit  
---
### Available Voices
See [main.go:77](https://github.com/BieHDC/annoyingtts/blob/master/main.go#L77)
