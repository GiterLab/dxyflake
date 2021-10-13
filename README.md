# dxyflake

duoxieyun distributed unique ID generator inspired by Twitter's Snowflake

## ID Format

    +------------------------------------------------------------------------------------------+
    | 1 Bit Unused | 41 Bit Timestamp | 5 Bit MachineID | 5 Bit ServiceID | 12 Bit Sequence ID |
    +------------------------------------------------------------------------------------------+

    41 bits for time in units of 10 msec (697 years)
     5 bits for a machine id (32 nodes)
     5 bits for a service id (32 services per node)
    12 bits for a sequence number (0 ~ 4095)

## Install

    go get github.com/GiterLab/dxyflake

## Usage

    package main

    import (
        "fmt"

        "github.com/GiterLab/dxyflake"
    )

    func main() {
        s := dxyflake.Settings{}
        s.Init(0, 0) // set mID & sID
        dxyid := dxyflake.NewDxyflake(s)

        id, err := dxyid.NextID()
        if err != nil {
            fmt.Println(err)
        }
        fmt.Println(id, dxyflake.Decompose(id))
    }

## License

The MIT License (MIT)

See [LICENSE](https://github.com/GiterLab/dxyflake/blob/master/LICENSE) for details.

## Reference

- [Snowflake](https://github.com/bwmarrin/snowflake)
- [Sonyflake](https://github.com/sony/sonyflake)
