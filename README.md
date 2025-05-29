# HuntingHash (holloman)

A fuzzy hashing for files and a special case for DNA. 

## Identifier Description

Identifer of /bin/ls on X86_64 j362e4894.23655e5f5a6264630807270e00000000

[order][xxhas(libmagic)].[128bits]
A Holloman Identifer for files is composed of three parts, two in the prefix and the suffix. They are seporated by a (.) dot. The prefix  has two parts, the first is a single char that specifies the order of the curve used to generate the curve. The order should be larger than the file's size, it't order-1 should be smaller than the file. The second part is a hex encoded 32bit unsigned integer derived from the file's first 60 characters (left justified) of the file's magic. The suffix is a 128 bit intger expressed in hex.

## The Process
A file is mapped onto a hilbert curve as 8bit greyscale pixels, this preserves the locality of bytes. The image is reduced. Inour case it is reduced to a 16 byte image by a lanczos three lobe resampler. This process appears to preserve enough of the origional file to "cluster" simular files. identifiers with the same prefix my be measured for distance by counting the number of bits different by applying a locical XOR to two 128 bit integers.

## DNA Encoding
A sumular procedure is applied for DNA, less the file magic. The prefix is simply the order of the hilbert curve and the suffix is a 128 bit integer. The pre-processor for DNA sequences reads fasta format files and sends the emcoded sequence to the server for mapping to a curve and resampleing. Encoding DNA (C,G,A,T) into greyscale pixels is described in the code. There are several ways to accomplish this encoding. We chose one based in the iChing, it seems to work well.


# LICENCE Review

To compare the Rick's Lifestyle Licence (RLL 1.0) with common open source licenses, I'll evaluate it against well-known open source licenses like the MIT License, GNU General Public License (GPL), Apache License 2.0, and BSD Licenses, focusing on key aspects such as permissions, restrictions, redistribution, and philosophy. The comparison will highlight how RLL 1.0 aligns with or diverges from the principles of open source software as defined by the Open Source Initiative (OSI).

## Definition of Open Source

The OSI defines open source software as software that can be freely used, modified, and distributed, with source code available, under licenses that meet criteria like free redistribution, no discrimination against persons or fields of endeavor, and no restriction on derivative works. Common open source licenses (MIT, GPL, Apache, BSD) adhere to these principles, but RLL 1.0 introduces unique constraints.

## Key Differences from Open Source Licenses

## Field-of-Use Restrictions:
RLL 1.0: Imposes significant restrictions by prohibiting Competing Use (e.g., AI training, substituting licensor’s products, or offering similar functionality). This violates OSI’s principle of no discrimination against fields of endeavor (e.g., restricting AI use or commercial competition).
Open Source Licenses: MIT, Apache, and BSD have no field-of-use restrictions, allowing use in any context, including commercial or AI applications. GPL allows all uses but requires copyleft compliance.

### License Key Protection:
RLL 1.0: Explicitly prohibits tampering with or bypassing license key functionality, suggesting a focus on controlling access to certain software features. This is uncommon in open source licenses, which typically assume full access to source code and functionality.
Open Source Licenses: No such restrictions exist, as open source licenses encourage full access to source code and functionality without technical barriers like license keys.

### Redistribution Constraints:
RLL 1.0: Redistribution is allowed but limited to non-competing uses, and all copies must include the license terms. This restriction on commercial redistribution for competing purposes is not OSI-compliant.
Open Source Licenses: MIT, Apache, and BSD allow unrestricted redistribution (with notice retention). GPL requires source code and GPL licensing for derivatives but doesn’t restrict commercial use.
### Patent Provisions:

RLL 1.0: Includes a patent license but terminates it if the user claims patent infringement, and the license is tied to Permitted Purposes.
Open Source Licenses: Apache explicitly grants patent licenses with a retaliation clause. MIT, GPL, and BSD typically lack explicit patent grants, though GPLv3 addresses patent issues indirectly.
Philosophy and OSI Compliance:

RLL 1.0: The license’s focus on protecting the licensor’s “lifestyle” and business interests (e.g., prohibiting competition or AI use) makes it a source-available license rather than open source. It fails OSI criteria due to restrictions on fields of endeavor and commercial use.
Open Source Licenses: MIT, GPL, Apache, and BSD are OSI-compliant, designed to maximize user freedom, with varying degrees of obligations (permissive vs. copyleft).

## Trademarks:

RLL 1.0: Explicitly restricts trademark use, which is unusual for open source licenses but aligns with source-available licenses that protect brand identity.
Open Source Licenses: Only Apache explicitly addresses trademarks; others (MIT, GPL, BSD) typically don’t, leaving trademark issues to separate legal considerations.

## 4. Alignment with Source-Available Licenses
RLL 1.0 is more akin to source-available licenses (e.g., Business Source License or Server Side Public License) than true open source licenses. These licenses provide access to source code but impose restrictions to protect the licensor’s commercial interests, such as limiting use in competing products or specific fields. RLL 1.0’s prohibition on AI training and competing uses mirrors these models but is uniquely tied to the licensor’s “lifestyle,” making it highly specific and restrictive.

## 5. Conclusion
RLL 1.0 is not an open source license under OSI standards due to its restrictions on competing uses, AI training, and license key tampering. It prioritizes the licensor’s control over commercial applications and specific use cases, making it a source-available license.
Open Source Licenses (MIT, GPL, Apache, BSD) are more permissive or copyleft-focused, designed to promote freedom and broad use without field-specific or competitive restrictions.

If you’re considering using software under RLL 1.0, it’s suitable for non-commercial, educational, or internal purposes but restricts commercial applications that might compete with the licensor’s offerings, unlike the flexibility of true open source licenses.
