project]
name = "browser-use"
description = "Make websites accessible for AI agents"
authors = [{ name = "Gregor Zunic" }]
version = "0.7.11"
readme = "README.md"
requires-python = ">=3.11,<4.0"
classifiers = [
    "Programming Language :: Python :: 3",
    "License :: OSI Approved :: MIT License",
    "Operating System :: OS Independent",
]
dependencies = [
    "aiohttp==3.12.15",
    "anyio>=4.9.0",
    "bubus>=1.5.6",
    "google-api-core>=2.25.0",
    "httpx>=0.28.1",
    "portalocker>=2.7.0,<3.0.0",
    "posthog>=3.7.0",
    "psutil>=7.0.0",
    "pydantic>=2.11.5",
    "pyobjc>=11.0; platform_system == 'darwin'",
    "python-dotenv>=1.0.1",
    "requests>=2.32.3",
    "screeninfo>=0.8.1; platform_system != 'darwin'",
    "typing-extensions>=4.12.2",
    "uuid7>=0.1.0",
    "authlib>=1.6.0",
    "google-genai>=1.29.0,<2.0.0",
    "openai>=1.99.2,<2.0.0",
    "anthropic>=0.68.1,<1.0.0",
    "groq>=0.30.0",
    "ollama>=0.5.1",
    "google-api-python-client>=2.174.0",
    "google-auth>=2.40.3",
    "google-auth-oauthlib>=1.2.2",
    "mcp>=1.10.1",
    "pypdf>=5.7.0",
    "reportlab>=4.0.0",
    "cdp-use>=1.4.0",
    "pyotp>=2.9.0",
    "html2text>=2025.4.15",
    "pillow>=11.2.1",
]
# google-api-core: only used for Google LLM APIs
# pyperclip: only used for examples that use copy/paste
# pyobjc: only used to get screen resolution on macOS
# screeninfo: only used to get screen resolution on Linux/Windows
# markdownify: used for page text content extraction for passing to LLM
# openai: datalib,voice-helpers are actually NOT NEEDED but openai produces noisy errors on exit without them TODO: fix
# rich: used for terminal formatting and styling in CLI
# click: used for command-line argument parsing
# textual: used for terminal UI

[project.optional-dependencies]
cli = [
    "rich>=14.0.0",
    "click>=8.1.8",
    "textual>=3.2.0",
]
aws = [
    "boto3>=1.38.45"
]
video = [
    "imageio[ffmpeg]>=2.37.0",
    "numpy>=2.3.2",
]
examples = [
    "agentmail==0.0.59",
    # botocore: only needed for Bedrock Claude boto3 examples/models/bedrock_claude.py
    "botocore>=1.37.23",
    "imgcat>=0.6.0",
    # "stagehand-py>=0.3.6",
    # "browserbase>=0.4.0",
    "langchain-openai>=0.3.26",
]
eval = [
    "lmnr[all]==0.7.17",
    "anyio>=4.9.0",
    "psutil>=7.0.0",
    "datamodel-code-generator>=0.26.0",
    "hyperbrowser==0.47.0",
    "browserbase==1.4.0",
]
all = [
    "browser-use[cli,examples,aws]",
]

# will prefer to use local source code checked out in ../../browser-use (if present) instead of pypi browser-use package
# [tool.uv.sources]
# bubus = { path = "../bubus", editable = true }


[project.urls]
Repository = "https://github.com/browser-use/browser-use"

[project.scripts]
browseruse = "browser_use.cli:main"
browser-use = "browser_use.cli:main"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"


[tool.codespell]
ignore-words-list = "bu,wit,dont,cant,wont,re-use,re-used,re-using,re-usable,thats,doesnt,doubleclick"
skip = "*.json"

[tool.ruff]
line-length = 130
fix = true

[tool.ruff.lint]
select = ["ASYNC", "E", "F", "FAST", "I", "PLE"]
ignore = ["ASYNC109", "E101", "E402", "E501", "F841", "E731", "W291"]  # TODO: determine if adding timeouts to all the unbounded async functions is needed / worth-it so we can un-ignore ASYNC109
unfixable = ["E101", "E402", "E501", "F841", "E731"]

[tool.ruff.format]
quote-style = "single"
indent-style = "tab"
line-ending = "lf"
docstring-code-format = true
docstring-code-line-length = 140
skip-magic-trailing-comma = false

[tool.pyright]
typeCheckingMode = "basic"
exclude = [".venv/", ".git/", "_pycache/", "./test.py", "./debug_.py", "private_example/", "debug/", "tests/scripts/", "tests/old/", "browser_use/dom/playground/"]
venvPath = "."
venv = ".venv"


[tool.hatch.build]
include = [
    "browser_use//*.py",
    "!browser_use//tests/*.py",
    "!browser_use//tests.py",
    "browser_use/agent/system_prompt.md",
    "browser_use/agent/system_prompt_no_thinking.md",
    "browser_use/agent/system_prompt_flash.md",
    "browser_use/py.typed",
    "browser_use/dom//*.js",
    "!tests//*.py",
    "!debug/*",
]

[tool.pytest.ini_options]
timeout = 300
asyncio_mode = "auto"
asyncio_default_fixture_loop_scope = "session"
asyncio_default_test_loop_scope = "session"
markers = [
    "slow: marks tests as slow (deselect with -m 'not slow')",
    "integration: marks tests as integration tests",
    "unit: marks tests as unit tests",
    "asyncio: mark tests as async tests",
]
testpaths = [
    "tests"
]
python_files = ["test_.py", "_test.py"]
addopts = "-svx --strict-markers --tb=short --dist=loadscope"
log_cli = true
log_cli_format = "%(levelname)-8s [%(name)s] %(message)s"
filterwarnings = [
    "ignore::pytest.PytestDeprecationWarning",
    "ignore::DeprecationWarning",
]
log_level = "DEBUG"


[tool.hatch.metadata]
allow-direct-references = true

[tool.uv]
# required-environments = [
#     "sys_platform == 'darwin' and platform_machine == 'arm64'",
#     "sys_platform == 'darwin' and platform_machine == 'x86_64'",
#     "sys_platform == 'linux' and platform_machine == 'x86_64'",
#     "sys_platform == 'linux' and platform_machine == 'aarch64'",
#     # "sys_platform == 'linux' and platform_machine == 'arm64'",  # no pytorch wheels available yet
#     "sys_platform == 'win32' and platform_machine == 'x86_64'",
#     # "sys_platform == 'win32' and platform_machine == 'arm64'",  # no pytorch wheels available yet
# ]
dev-dependencies = [
    "ruff>=0.11.2",
    "tokencost>=0.1.16",
    "build>=1.2.2",
    "pytest>=8.3.5",
    "pytest-asyncio>=1.0.0",
    "pytest-httpserver>=1.0.8",
    "fastapi>=0.115.8",
    "inngest>=0.4.19",
    "uvicorn>=0.34.0",
    "ipdb>=0.13.13",
    "pre-commit>=4.2.0",
    "codespell>=2.4.1",
    "pyright>=1.1.403",
    "ty>=0.0.1a1",
    "pytest-xdist>=3.7.0",
    "lmnr[all]==0.7.17",
    # "pytest-playwright-asyncio>=0.7.0",  # not actually needed I think
    "pytest-timeout>=2.4.0",
    "pydantic_settings>=2.10.1"
]