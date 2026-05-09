package translate

const playlistTranslatePrompt = `
# Video Translation Task

You are a professional video translator proficient in multiple languages with automatic language detection capabilities. Please complete the following translation task.

## Task Requirements:
1. **Automatically detect** the source language of the content to be translated
2. Translate the content into each language specified in target_language
3. If the original data's language is not in target_language, do not include the original data in the returned results

**target_language**: %s

## Content to Translate:
title: %s
description: %s
tags: %s
seo_title: %s
seo_description: %s
seo_keywords: %s

## Output Format:
You MUST return the following JSON structure without any additional content:
` + "```json\n" + `[
  {
    "language": "[value from target_language, keep as is]",
    "title": "[translated title]",
    "description": "[translated description]",
    "tags": "[translated tags]",
    "seo_title": "[translated seo_title]",
    "seo_description": "[translated seo_description]",
    "seo_keywords": "[translated seo_keywords]"
  }
]
` + "```" + `

## Translation Guidelines:
1. Maintain the original style, tone, and professionalism
2. For SEO-related fields (seo_title, seo_description, seo_keywords), consider SEO best practices for the target language

## Important Constraints:
- Return only parseable JSON array wrapped with []
- Even if there is only one item, return it as a JSON array
- The language field in translated data must match exactly with the target_language value (character-level matching)
- Accurately auto-detect the original data's language field; if it's not in target_language, do not return it
- Do not output any additional explanations, notes, or markdown formatting
`

const pageTranslatePrompt = `# Video Translation Task

You are a professional video translator proficient in multiple languages with automatic language detection capabilities. Please complete the following translation task.

## Task Requirements:

1. **Automatically detect** the source language of the content to be translated
2. Translate the content into each language specified in target_language
3. If the original data's language is not in target_language, do not include the original data in the returned results

**target_language**: %s

## Content to Translate:

name: %s;
title: %s;
description: %s;
keywords: %s;

## Output Format:

You MUST return the following JSON structure without any additional content:
` + "```json\n" + `[
  {
    "language": "[value from target_language, keep as is]",
	"name": "[translated name]",
    "title": "[translated title]",
    "description": "[translated description]",
    "keywords": "[translated keywords]"
  }
]
` + "```" + `

## Translation Guidelines:

Maintain the original style, tone, and professionalism

## Important Constraints:

- Return only parseable JSON array wrapped with []
- Return empty values for fields that are empty in the original
- Even if there is only one item, return it as a JSON array
- The language field in translated data must match exactly with the target_language value (character-level matching)
- Accurately auto-detect the original data's language field; if it's not in target_language, do not return it
- Do not output any additional explanations, notes, or markdown formatting`
