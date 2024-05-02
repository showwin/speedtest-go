# Issue 192

<img width="1074" alt="SpeedTest-Go" src="https://github.com/showwin/speedtest-go/assets/30739857/203da134-77f7-46a1-bed2-2df3c43ffc7c">

1. Use welford alg to quickly calculate standard deviation and mean.
2. (a)The welford method integrated moving window feature, This allows us to ignore early data with excessive volatility.
3. (b)The welford method integrated moving Average feature, compare with 2a, this causes early data to be diluted gradually rather than immediately.
4. Use the coefficient of variation(c.v) to reflect the confidence of the test result datasets.
5. When the c.v is less than 0.05, we terminate this test.

Is there a better way?