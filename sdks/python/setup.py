import os
from setuptools import setup, find_packages

with open(os.path.join(os.path.dirname(__file__), 'README.md'), encoding='utf-8') as f:
    long_description = f.read()

setup(
    name="tkngate",
    version="1.0.1",
    description="Drop-in wrapper for the OpenAI Python SDK to route traffic through the Tkngate proxy.",
    long_description=long_description,
    long_description_content_type="text/markdown",
    author="Tkngate Contributors",
    author_email="hello@tkngate.dev",
    packages=find_packages(),
    install_requires=[
        "openai>=1.0.0"
    ],
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: Apache Software License",
        "Operating System :: OS Independent",
    ],
    python_requires='>=3.7',
)
