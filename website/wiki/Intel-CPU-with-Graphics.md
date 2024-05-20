| Gen | Name                | Year | Core                         | Xeon       | Pentium                | Celeron                    |
|-----|---------------------|------|------------------------------|------------|------------------------|----------------------------|
| 5.5 | Westmere (Ironlake) | 2010 | i3/5/7-xxx                   |            | (G/P)6000, U5000       | P4000, U3000               |
| 6   | Sandy Bridge        | 2011 | i3/5/7-2000                  | E3-1200    | (B)900, (G)800, (G)600 | (B)800, (B)700, G500, G400 |
| 7   | Ivy Bridge          | 2012 | i3/5/7-3000                  | E3-1200 v2 | (G)2000, A1018         | G1600, 1000, 900           |
| 7   | Bay Trail (SoC)     | 2013 |                              |            | J2000, N3500, A1020    | J1000, N2000               |
| 7.5 | Haswell             | 2013 | i3/5/7-4000                  | E3-1200 v3 | (G)3000                | G1800, 2000                |
| 8   | Broadwell           | 2014 | i3/5/7-5000                  | E3-1200 v4 | 3800                   | 3700, 3200                 |
| 8   | Braswell (SoC)      | 2013 |                              |            | (J/N)37x0              | (J/N)30x0, N31x0           |
| 9   | Skylake             | 2015 | i3/5/7-6000                  |            | (G)4000                | 3900, 3800                 |
| 9   | Apollo Lake (SoC)   | 2016 |                              | E3-1x00 v5 | (J/N)4xxx              | (J/N)3xxx                  |
| 9.5 | Kaby Lake           | 2016 | m3/i3/5/7-7000               |            | (G)4000                | (G)3900, 3800              |
| 9.5 | Coffee Lake         | 2017 | i3/5/7/9-8000, i3/5/7/9-9000 | E-2x00     | (G)5xxx                | (G)49xx                    |
| 9.5 | Gemini Lake (SoC)   | 2017 |                              | E3-1x00 v6 | (J/N)5xxx              | (J/N)4xxx                  |
| 9.5 | Whiskey Lake        | 2018 | i3/5/7-8000U                 |            |                        |                            |
| 9.5 | Amber Lake          | 2018 |                              |            |                        |                            |
| 9.5 | Comet Lake          | 2019 | i3/5/7-10xxx                 | W-108xxM   | (G)6x00                | G59x0                      |
| 11  | Ice Lake            | 2019 | i3/5/7-10xx(N)Gx             |            |                        |                            |
| 11  | Lakefield           | 2020 | ???                          |            |                        |                            |
| 11  | Elkhart/Jasper Lake | 2021 |                              |            | ???                    | ???                        |
| 12  | Tiger Lake          | 2020 | i3/5/7-11xx(N)Gx             | W-11xxxM   | (G)7xxx                | (G)6xxx                    |

Gen - Graphic generation, not related to CPU generation

- **Sandy Bridge** (2011): decoding/encoding for AVC/H.264
- **Haswell** (2013): decoding/encoding for MJPEG
- **Skylake** (2015): decoding/encoding for HEVC/H.265
- [i965-va-driver](https://packages.debian.org/ru/sid/i965-va-driver) from **Westmere** to **Coffee Lake**
- [intel-media-va-driver](https://packages.debian.org/sid/intel-media-va-driver) from **Broadwell**

## Useful links

- https://en.wikipedia.org/wiki/Intel_Graphics_Technology#Capabilities_(GPU_hardware)
- https://en.wikipedia.org/wiki/Intel_Quick_Sync_Video#Hardware_decoding_and_encoding
- https://www.intel.com/content/www/us/en/developer/tools/openvino-toolkit/system-requirements.html