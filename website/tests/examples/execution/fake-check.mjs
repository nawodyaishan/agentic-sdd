// Inert fixture: a fake check command for the execution example test.
// It prints fixed text and exits 1. It reads no environment variable, opens no
// socket, and writes no file. The test runs it only inside a disposable copy.
process.stdout.write('ok   example/pkg/fast 0.010s\n');
process.stderr.write('--- FAIL: ExampleSummary (0.00s)\n');
process.exitCode = 1;
