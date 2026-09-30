# Copyright 2026 Scaleway
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

from qall_registry_client.v1.objects import Block
from qall_registry_client.v1.client import RegistryClient


@dataclass
class GetResult:
    blocks: list[Block]
    downloaded_blocks: list[str]
    total_blocks_count: int


def get(
    hashes: list[str],
    codec_check: callable,
    codec_decode: callable,
    client: Optional[RegistryClient],
    next_get: Optional[callable] = None,
    fallback_get: Optional[callable] = None,
) -> GetResult:

    if not hashes:
        return GetResult(blocks=[], downloaded_blocks=[], total_blocks_count=0)

    results: dict[str, Block] = {}
    downloaded_blocks: list[str] = []
    visited_hashes: set[str] = set()

    def _fetch(current_hashes: list[str]) -> None:
        hashes_to_fetch = []

        for h in current_hashes:
            if h in visited_hashes:
                continue
            visited_hashes.add(h)
            hashes_to_fetch.append(h)

        if hashes_to_fetch:
            fetched_blocks = client.get(hashes_to_fetch)

            if (not fetched_blocks or len(fetched_blocks) == 0) and fallback_get:
                fetched_blocks = fallback_get(hashes_to_fetch)

            for b in fetched_blocks:
                if not codec_check(b.data, b.hash):
                    raise ValueError(
                        f"Cryptographic integrity violation: block data verification "
                        f"failed for expected hash '{b.hash}'."
                    )

                downloaded_blocks.append(b.hash)
                results[b.hash] = b

        if next_get:
            additional_hashes: set[str] = set()

            for h in current_hashes:
                block = results.get(h)
                if not block:
                    continue

                try:
                    decoded = codec_decode(block.data)

                    if decoded:
                        next_hashes = next_get(decoded)
                        if isinstance(next_hashes, str):
                            additional_hashes.add(next_hashes)
                        if isinstance(next_hashes, list):
                            additional_hashes.update(next_hashes)
                except Exception:
                    continue

            if additional_hashes:
                _fetch(list(additional_hashes))

    _fetch(hashes)

    sorted_blocks = [results[h] for h in sorted(results.keys())]

    return GetResult(
        blocks=sorted_blocks,
        downloaded_blocks=downloaded_blocks,
        total_blocks_count=len(sorted_blocks),
    )
