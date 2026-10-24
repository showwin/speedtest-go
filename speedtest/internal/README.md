# Rate meter

`Meter` turns cumulative byte counts, sampled every 50 ms, into the rate a test reports and decides when the test may stop.

1. The rate is bytes over time across the last two thirds of the elapsed time. TCP ramp-up falls out of the window as the test runs, so the result converges on the link's steady rate without a ramp-up detector.
2. The rate has settled when two standard errors and the difference between the window's two halves are both within 1% of it. The standard error is the Newey-West estimate over the 50 ms byte increments (Bartlett weights, square root of n lags), which accounts for successive throughput samples being correlated.
3. A test stops once the rate has settled, no earlier than 4 seconds, and otherwise at its capture time (15 seconds by default). A zero rate never settles.

The live callbacks, the final result and the stopping decision all read the same meter. Upload feeds it the bytes the server's TCP stack has acknowledged where the platform reports them (Linux, macOS and Windows), and confirmed request bytes elsewhere.
