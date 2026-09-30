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
import os

import pytest
import tempfile
import random
import string
import concurrent.futures

from pathlib import Path
from typing import List, Dict

import qall_registry_client.v1 as registry

from qall_registry_client.v1 import (
    GrpcRegistryClient,
    Payload,
    Node,
    Block,
    Tag,
    RegistryClient,
    LocalBlockstore,
    DEFAULT_CODEC,
)


class _MockMemoryRegistryServer(RegistryClient):
    def __init__(self):
        self.blocks: Dict[str, bytes] = {}
        self.tags: Dict[str, Tag] = {}

    def check(self, hashes: List[str]) -> List[str]:
        return [h for h in hashes if h not in self.blocks]

    def push(self, blocks: List[Block]) -> int:
        written = 0
        for b in blocks:
            if b.hash not in self.blocks:
                self.blocks[b.hash] = b.data
                written += 1
        return written

    def get(self, hashes: List[str]) -> List[Block]:
        return [Block(hash=h, data=self.blocks[h]) for h in hashes if h in self.blocks]

    def tag(self, tag: Tag) -> str:
        uri = f"{tag.name}:{tag.version}"
        self.tags[uri] = tag
        if tag.version != "latest":
            self.tags[f"{tag.name}:latest"] = Tag(
                name=tag.name, version="latest", root_hash=tag.root_hash
            )
        return uri

    def resolve(self, name: str, version: str) -> str:
        uri = f"{name}:{version}"
        if uri in self.tags:
            return self.tags[uri].root_hash
        return ""

    def untag(self, name: str, version: str) -> bool:
        uri = f"{name}:{version}"
        if uri in self.tags:
            del self.tags[uri]
            return True
        return False

    def list_tags(self, name: str) -> List[Tag]:
        return [tag for tag in self.tags.values() if tag.name == name]


@pytest.fixture
def test_env():
    server_url = os.getenv("QALL_REGISTRY_URL")

    with tempfile.TemporaryDirectory() as tmp_dir:
        root_path = Path(tmp_dir)
        store = LocalBlockstore(root_dir=root_path)

        if server_url:
            server = GrpcRegistryClient(url=server_url)
        else:
            server = _MockMemoryRegistryServer()

        yield store, server, DEFAULT_CODEC

        if hasattr(server, "close"):
            server.close()


def gen_random_code() -> str:
    nonce = "".join(random.choices(string.ascii_letters + string.digits, k=8))
    return f"def run_{nonce}():\n    return 'quantum_boost'"


def test_pipeline_push_with_differential_caching(test_env):
    store, server, codec = test_env

    payload = Payload(
        code=gen_random_code(),
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests", "pydantic", "numpy"],
        },
    )

    p_blk = payload.to_block(codec)

    node = Node(payload_hash=p_blk.hash)
    n_blk = node.to_block(codec)

    res1 = registry.push(
        name="vqe_chemistry",
        version="v1.0.0",
        payloads=[payload],
        nodes=[node],
        root_hash=n_blk.hash,
        client=server,
        store=store,
        codec=codec,
    )

    assert res1.total_blocks_count == 2
    assert len(res1.uploaded_blocks) == 2 or len(res1.uploaded_blocks) == 1
    assert res1.skipped_blocks_by_local_cache_count == 0
    assert server.resolve("vqe_chemistry", "latest") == n_blk.hash
    assert registry.resolve(server, "vqe_chemistry", "latest") == n_blk.hash
    assert registry.resolve(server, "vqe_chemistry") == n_blk.hash
    assert registry.resolve(server, "vqe_chemistry:latest") == n_blk.hash

    res2 = registry.push(
        name="vqe_chemistry",
        version="v1.0.1",
        payloads=[payload],
        nodes=[node],
        root_hash=n_blk.hash,
        client=server,
        store=store,
        codec=codec,
    )

    assert res2.total_blocks_count == 2
    assert len(res2.uploaded_blocks) == 0
    assert res2.skipped_blocks_by_local_cache_count == 2


def test_pull_and_complete_object_deserialization(test_env):
    store, server, codec = test_env

    payload = Payload(
        code=gen_random_code(),
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )

    p_blk = payload.to_block(codec)
    node = Node(payload_hash=p_blk.hash)
    n_blk = node.to_block(codec)

    server.push([p_blk, n_blk])
    server.tag(Tag(name="analog_algo", version="v2.0", root_hash=n_blk.hash))

    assert not store.has(n_blk.hash)
    pull_result = registry.pull(
        client=server, name="analog_algo", version="v2.0", store=store, codec=codec
    )

    assert pull_result.total_blocks_count == 2
    assert len(pull_result.downloaded_blocks) == 2

    assert isinstance(pull_result.nodes[0], Node)
    assert isinstance(pull_result.payloads[0], Payload)


def test_get_with_corrupted_server_response_must_fail(test_env):
    store, server, codec = test_env

    if not isinstance(server, _MockMemoryRegistryServer):
        pytest.skip("This test is only relevant for the in-memory mock server")

    valid_payload = Payload(
        code=gen_random_code(),
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    v_blk = valid_payload.to_block(codec)

    server.blocks[v_blk.hash] = b"CORRUPTED_BIN_DATA_INJECTED_DURING_TRANSIT"

    with pytest.raises(ValueError) as exc_info:
        registry.get(client=server, hashes=[v_blk.hash], store=store, codec=codec)

    assert "Cryptographic integrity violation" in str(exc_info.value)
    assert not store.has(v_blk.hash)


def test_compromised_local_blockstore_detection(test_env):
    store, server, codec = test_env

    payload = Payload(
        code="def target(): print('ok')",
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    blk = payload.to_block(codec)
    store.put(blk.hash, blk.data)

    shard_dir = store._blocks_dir / blk.hash[:2]
    file_path = shard_dir / blk.hash[2:]

    assert file_path.exists()
    file_path.write_bytes(
        b'{"code": "def target(): os.system(\'malicious_cmd\')", "corrupted": true}'
    )

    server.push([blk])
    server.tag(Tag(name="secure_wf", version="latest", root_hash=blk.hash))

    pull_result = registry.pull(
        client=server, name="secure_wf", store=store, codec=codec
    )

    assert len(pull_result.payloads) == 0


def test_pipeline_partial_modification_delta_push(test_env):
    store, server, codec = test_env
    ctx_name = "partial_update_wf"

    payload_v1 = Payload(
        code="def run(): return 1",
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    p1_blk = payload_v1.to_block(codec)
    node_v1 = Node(payload_hash=p1_blk.hash)
    n1_blk = node_v1.to_block(codec)

    registry.push(
        name=ctx_name,
        version="v1.0.0",
        root_hash=n1_blk.hash,
        payloads=[payload_v1],
        nodes=[node_v1],
        client=server,
        store=store,
        codec=codec,
    )

    payload_v2 = Payload(
        code="def run(): return 2  # update code !",
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    p2_blk = payload_v2.to_block(codec)

    node_v2 = Node(payload_hash=p2_blk.hash)
    n2_blk = node_v2.to_block(codec)

    res = registry.push(
        name=ctx_name,
        version="v2.0.0",
        root_hash=n2_blk.hash,
        payloads=[payload_v2],
        nodes=[node_v2],
        client=server,
        store=store,
        codec=codec,
    )

    assert res.total_blocks_count == 2
    assert len(res.uploaded_blocks) == 1 or len(res.uploaded_blocks) == 0
    assert res.skipped_blocks_by_local_cache_count == 0


def test_blockstore_enforces_max_size_limit(test_env):
    store, _, _ = test_env

    massive_data = b"X" * (20 * 1024 * 1024)

    with pytest.raises(ValueError) as exc_info:
        store.put(hash="fake_hash_for_massive_payload", data=massive_data)

    assert "Payload size exceeds maximum allowed registry bounds" in str(exc_info.value)


def test_tag_latest_overwriting_lifecycle(test_env):
    store, server, codec = test_env
    wf_name = "rollover_wf"

    p_v1 = Payload(
        code="def root(): return 'A'",
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    node_v1 = Node(payload_hash=p_v1.hash)
    n1_blk = node_v1.to_block(codec)
    server.push([p_v1.to_block(codec), n1_blk])
    server.tag(Tag(name=wf_name, version="v1.0.0", root_hash=n1_blk.hash))

    assert server.resolve(wf_name, "latest") == n1_blk.hash

    p_v2 = Payload(
        code="def root(): return 'B'",
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    node_v2 = Node(payload_hash=p_v2.hash)
    n2_blk = node_v2.to_block(codec)
    server.push([p_v2.to_block(codec), n2_blk])
    server.tag(Tag(name=wf_name, version="v2.0.0", root_hash=n2_blk.hash))

    assert server.resolve(wf_name, "latest") == n2_blk.hash

    p_v1_fix = Payload(
        code="def root(): return 'A_fixed'",
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    node_v3 = Node(payload_hash=p_v1_fix.hash)
    n3_blk = node_v3.to_block(codec)
    server.push([p_v1_fix.to_block(codec), n3_blk])
    server.tag(Tag(name=wf_name, version="v1.0.1", root_hash=n3_blk.hash))

    assert server.resolve(wf_name, "latest") == n3_blk.hash


def test_workflow_complete_lifecycle_stateless(test_env):
    store, server, codec = test_env
    wf_name = "vqe_stateless_pipeline"

    payload = Payload(
        code=gen_random_code(),
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )

    p_blk = payload.to_block(codec)
    node = Node(payload_hash=p_blk.hash)
    n_blk = node.to_block(codec)

    push_res = registry.push(
        name=wf_name,
        version="v1.0.0",
        payloads=[payload],
        nodes=[node],
        root_hash=n_blk.hash,
        client=server,
        store=store,
        codec=codec,
    )
    assert push_res.total_blocks_count == 2
    assert len(push_res.uploaded_blocks) == 2 or len(push_res.uploaded_blocks) == 1

    resolved_root = registry.resolve(client=server, name=wf_name, version="v1.0.0")
    assert resolved_root == n_blk.hash

    get_res = registry.get(
        client=server, hashes=[resolved_root], recursive=True, codec=codec
    )

    assert get_res.total_blocks_count == 2
    assert len(get_res.downloaded_blocks) == 2
    assert any(b.hash == p_blk.hash for b in get_res.blocks)

    with tempfile.TemporaryDirectory() as worker_tmp:
        worker_store = LocalBlockstore(root_dir=Path(worker_tmp))

        pull_res = registry.pull(
            client=server,
            name=wf_name,
            version="latest",
            store=worker_store,
            codec=codec,
        )

        assert pull_res.total_blocks_count == 2
        assert len(pull_res.downloaded_blocks) == 2
        assert len(pull_result_nodes := pull_res.nodes) == 1
        assert len(pull_result_payloads := pull_res.payloads) == 1

        assert isinstance(pull_result_nodes[0], Node)
        assert isinstance(pull_result_payloads[0], Payload)


def test_workflow_tag_management_stateless(test_env):
    store, server, codec = test_env
    wf_name = "tag_admin_wf"

    payload = Payload(
        code=gen_random_code(),
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    p_blk = payload.to_block(codec)
    node = Node(payload_hash=p_blk.hash)
    n_blk = node.to_block(codec)
    server.push([p_blk, n_blk])

    new_tag = Tag(name=wf_name, version="v3.1.4", root_hash=n_blk.hash)
    tag_success = registry.tag(client=server, tag=new_tag)
    assert tag_success is not None
    assert tag_success == f"{wf_name}:v3.1.4"

    all_tags = registry.list_tags(client=server, name=wf_name)
    assert len(all_tags) == 2
    assert any(t.version == "v3.1.4" for t in all_tags)
    assert any(t.version == "latest" for t in all_tags)

    untag_success = registry.untag(client=server, name=wf_name, version="v3.1.4")
    assert untag_success is True

    remaining_tags = registry.list_tags(client=server, name=wf_name)
    assert len(remaining_tags) == 1
    assert remaining_tags[0].version == "latest"

    res = registry.resolve(client=server, name=wf_name, version="v3.1.4")
    assert res == ""


def test_concurrent_furious_push_race_condition(test_env):
    store, server, codec = test_env

    payload = Payload(
        code=gen_random_code(),
        code_format="raw",
        environment={
            "image": "docker-proxy.internal.scaleway.com/python:3.11-slim",
            "requirements": ["requests pydantic numpy"],
        },
    )
    blk = payload.to_block(codec)

    unconfirmed_hashes = [blk.hash]

    def simulate_concurrent_user():
        missing = server.check(unconfirmed_hashes)
        if missing:
            return server.push([blk])
        return 0

    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        futures = [executor.submit(simulate_concurrent_user) for _ in range(10)]
        results = [f.result() for f in concurrent.futures.as_completed(futures)]

    assert all(isinstance(res, int) for res in results)
