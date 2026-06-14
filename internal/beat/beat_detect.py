#!/usr/bin/env python3
"""
Audio energy analysis from raw float32 audio via stdin.
Outputs JSON lines with bass energy for visualization.
"""

import sys
import struct
import math
import json

SAMPLE_RATE = 44100
HOP_SIZE = 512
FRAME_SIZE = 1024

# Lowpass filter for bass (~200Hz)
LP_B = [0.0675, 0.2024, 0.2024, 0.0675]
LP_A = [1.0, -0.7821, 0.3492, -0.0561]


def lowpass(samples, state):
    if state is None:
        state = [0.0] * 6
    out = []
    x = state[:]
    for s in samples:
        y = (LP_B[0]*s + LP_B[1]*x[0] + LP_B[2]*x[1] + LP_B[3]*x[2]
             - LP_A[1]*x[3] - LP_A[2]*x[4] - LP_A[3]*x[5])
        out.append(y)
        x = [s, x[0], x[1], y, x[3], x[4]]
    return out, x


def main():
    state = None
    buf = bytearray()
    frame = 0

    while True:
        chunk = sys.stdin.buffer.read(HOP_SIZE * 4)
        if not chunk:
            break
        buf.extend(chunk)

        while len(buf) >= FRAME_SIZE * 4:
            raw = buf[:FRAME_SIZE * 4]
            buf = buf[FRAME_SIZE * 4:]

            samples = struct.unpack(f'{FRAME_SIZE}f', raw)
            bass, state = lowpass(samples, state)
            energy = math.sqrt(sum(s*s for s in bass) / len(bass))
            now_ms = int(frame * HOP_SIZE / SAMPLE_RATE * 1000)

            sys.stdout.write(json.dumps({
                "type": "beat",
                "timestamp": now_ms,
                "energy": round(energy, 4),
                "is_onset": False,
            }) + "\n")
            sys.stdout.flush()
            frame += 1


if __name__ == "__main__":
    main()
