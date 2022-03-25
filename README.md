Streamzeug has been developed as a multi use streaming broadcast tool 
with a focus on reliable delivery of signals accross lossy networks.

It has been in in-house production use since 2021 retransmitting 24/7
TV-Channels at multiple handover points.

License for all code in repository is GPL-3 or later except included code 
in dektec/DTAPI, vectorio and include_srt folders. Specific licenses apply
and are supplied in those folders.

## Current feature set:  
- RIST input  
- ASI  output via Dektec devices  
- SRT  output  
- UDP  output  
- RTP  output  
- InfluxDB stats reporting  

## Future extensions:  
- RIST output  
- SRT  input  
- UDP  input  
- RTP  input  
- Failover between 2 active sources  
- TR 101 290 checking  

## Dependencies:  
- Golang  
- C++  
- DektecAPI (object files included in project)  
- libRIST (build as static linked sub project via meson build)  
- libSRT  (build as static linked sub project via meson build)  
  
## Building:
Streamzeug can be build either via the toplevel Makefile or via meson.
The makefile build requires libRIST and SRT to be present.
The meson build automatically builds libRIST and SRT as subprojects
and links those statically.  
To build via meson:
```
meson -c build  
cd build  
ninja  
```

## Special thanks
- Gijs Peskens

For developing Streamzeug and pushing the envelope in the [librist](https://code.videolan.org/rist/librist) project. Best of luck on your future endavours! o/

# Copyright © 2021-2022 in2ip B.V. / ODMedia B.V.
