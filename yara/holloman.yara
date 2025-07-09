import "holloman"

rule osx_bin_ls
{
    condition:
        holloman.hh128(0, filesize) == "ja90e5fa5.07510b05123317430103250e00000000"
}
