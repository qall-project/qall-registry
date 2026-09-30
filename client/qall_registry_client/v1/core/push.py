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

from qall_registry_client.v1.objects import Tag, Block
from qall_registry_client.v1.client import RegistryClient


@dataclass(frozen=True)
class PushResult:
    tag_uri: str
    total_blocks_count: int
    uploaded_blocks: list[str]


def push(
    name: str,
    version: str,
    blocks: list[Block],
    root_hash: str,
    client: RegistryClient,
) -> PushResult:
    all_hashes = [b.hash for b in blocks]

    missing_hashes = client.check(all_hashes)

    blocks_to_upload = [b for b in blocks if b.hash in missing_hashes]
    if blocks_to_upload:
        client.push(blocks_to_upload)

    tag = Tag(name=name, version=version, root_hash=root_hash)
    uri = client.tag(tag)

    return PushResult(
        tag_uri=uri,
        total_blocks_count=len(blocks),
        uploaded_blocks=[b.hash for b in blocks_to_upload],
    )
