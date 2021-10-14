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
        "os"

        "github.com/GiterLab/dxyflake"
    )

    func main() {
        s := dxyflake.Settings{}
        s.Init(0, 0) // set mID & sID
        dxyid := dxyflake.NewDxyflake(s)

        id, err := dxyid.NextID()
        if err != nil {
            fmt.Println(err)
            os.Exit(0)
        }
        fmt.Println(id, id.LeadingZerosString(19), dxyflake.Decompose(id))
        idBase64 := id.Base64()
        id, err = dxyflake.ParseBase64(idBase64)
        if err != nil {
            fmt.Println(err)
            os.Exit(0)
        }
        fmt.Println(idBase64, "-->", id)

        // 19 MAX
        fmt.Println("9223372036854775807", dxyflake.Decompose(dxyflake.ID(9223372036854775807))) // 697 years
    }

    // Output:
    //
    // 475370495148032 0000475370495148032 map[id:475370495148032 machine-id:0 msb:0 sequence:0 service-id:0 time:113337158]
    // NDc1MzcwNDk1MTQ4MDMy --> 475370495148032
    // 9223372036854775807 map[id:9223372036854775807 machine-id:31 msb:0 sequence:4095 service-id:31 time:2199023255551]

## License

The MIT License (MIT)

See [LICENSE](https://github.com/GiterLab/dxyflake/blob/master/LICENSE) for details.

## Reference

- [Snowflake](https://github.com/bwmarrin/snowflake)
- [Sonyflake](https://github.com/sony/sonyflake)
