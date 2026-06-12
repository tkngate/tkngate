const { Socket } = require('dgram');
const { setServers } = require('dns');
const fs = require('fs');
const pdf = require('pdf-parse');
const { jaccard, PRSenseDetector } = require('prsense');
const { useOptimistic } = require('react');
const { inflateRawSync } = require('zlib');

let dataBuffer = fs.readFileSync('Newproject.pdf');

pdf(dataBuffer).then(function(data) {
    // number of pages
    console.log("Pages:", data.numpages);
    // number of rendered pages
    console.log("Info:", data.info);
    // PDF text
    console.log("Text:");
    console.log(data.text);
}).catch(function(error){
    console.log("Error:", error);
});

