from setuptools import setup, find_packages

setup(
    name="tkngate",
    version="1.0.0",
    description="Drop-in wrapper for the OpenAI Python SDK to route traffic through the Tkngate proxy.",
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
