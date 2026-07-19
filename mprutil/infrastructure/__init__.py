from mprutil.infrastructure.bson_parser import BsonParser
from mprutil.infrastructure.mpr_base import MprReader
from mprutil.infrastructure.mpr_v1 import MprV1Reader
from mprutil.infrastructure.mpr_v2 import MprV2Reader


def open_mpr(path: str) -> MprReader:
    """自动检测 v1/v2 并打开"""
    import os

    if path.endswith(".mpr"):
        # 尝试 v2: mprcontents/ 优先
        base = path.rsplit(".mpr", 1)[0]
        mpr_contents = os.path.join(os.path.dirname(path), "mprcontents")
        if os.path.isdir(mpr_contents):
            return MprV2Reader(path)
        return MprV1Reader(path)

    # 直接指向 mprcontents/ 目录
    if os.path.isdir(path):
        return MprV2Reader(path)

    raise ValueError(f"unknown MPR path: {path}")


__all__ = [
    "MprReader",
    "MprV1Reader",
    "MprV2Reader",
    "BsonParser",
    "open_mpr",
]
