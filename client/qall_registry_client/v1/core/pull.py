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
from qall_registry_client.v1.core.resolve import resolve
from qall_registry_client.v1.core.get import get, GetResult


@dataclass
class PullResult:
    blocks: list[Block]
    total_blocks_count: int
    root_hash: str
    error: Optional[str] = None


def pull(
    client: RegistryClient,
    codec_check: Optional[callable],
    codec_decode: Optional[callable],
    name: str,
    version: Optional[str] = None,
    next_get: Optional[callable] = None,
    fallback_get: Optional[callable] = None,
) -> PullResult:
    root_hash = resolve(client, name, version)

    get_res = get(
        [root_hash],
        codec_check=codec_check,
        codec_decode=codec_decode,
        client=client,
        next_get=next_get,
        fallback_get=fallback_get,
    )

    return _getresult_to_pullresult(get_res=get_res, root_hash=root_hash)


def _getresult_to_pullresult(
    get_res: GetResult,
    root_hash: str,
) -> PullResult:

    return PullResult(
        blocks=get_res.blocks,
        downloaded_blocks=get_res.downloaded_blocks,
        total_blocks_count=get_res.total_blocks_count,
        root_hash=root_hash,
        error=None,
    )
