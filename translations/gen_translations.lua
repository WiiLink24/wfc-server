local http_request = require("http.request")
local utils = require("./utils")

local SHEETS_CSV_URL =
    [[https://docs.google.com/spreadsheets/d/1kas1J6RcIePcaRRxtTluPZm8C8kydpaoQBtRg15M-zM/export?format=tsv&gid=1517055494#gid=1517055494]]

local SHEET_LANG_TO_WWFC_LANG = {
    Japanese = "LangJapanese",
    English = "LangEnglish",
    German = "LangGerman",
    -- French = "LangFrench", -- We only have EU French
    Spanish = "LangSpanish",
    Italian = "LangItalian",
    Dutch = "LangDutch",
    ["Chinese (Simplified)"] = "LangSimpChinese",
    -- LangTradChinese
    Korean = "LangKorean",

    -- LangEnglishEU -- We only have American English
    French = "LangFrenchEU",
    -- LangSpanishEU -- We only have Americas Spanish

    -- Custom Languages:
    Czech = "LangCzech",
    Norwegian = "LangNorwegian",
    Russian = "LangRussian",
    Arabic = "LangArabic",
    Turkish = "LangTurkish",
    Finnish = "LangFinnish",
    Portuguese = "LangPortugueseEU",
}

local ORDERED_MESSAGES = {
    "WWFCMsgUnknownLoginError",
    "WWFCMsgDolphinSetupRequired",
    "WWFCMsgProfileBannedTOS",
    "WWFCMsgProfileBannedTOSNow",
    "WWFCMsgProfileRestricted",
    "WWFCMsgProfileRestrictedNow",
    "WWFCMsgProfileRestrictedCustom",
    "WWFCMsgProfileRestrictedNowCustom",
    "WWFCMsgKickedGeneric",
    "WWFCMsgKickedModerator",
    "WWFCMsgKickedRoomHost",
    "WWFCMsgKickedCustom",
    "WWFCMsgConsoleMismatch",
    "WWFCMsgConsoleMismatchDolphin",
    "WWFCMsgProfileIDInvalid",
    "WWFCMsgProfileIDInUse",
    "WWFCMsgPayloadInvalid",
    "WWFCMsgInvalidELO",
}

local ORDERED_LANGUAGES = {
    "Japanese",
    "English",
    "German",
    "Spanish",
    "Italian",
    "Dutch",
    "Chinese (Simplified)",
    "Korean",

    -- Custom
    "Czech",
    "Norwegian",
    "Russian",
    "Arabic",
    "Turkish",
    "Finnish",

    -- EU
    "French",
    -- Custom
    "Portuguese",
}

local headers, stream = assert(http_request.new_from_uri(SHEETS_CSV_URL):go())
local body = assert(stream:get_body_as_string())
if headers:get(":status") ~= "200" then
    error(body)
end

-- local infd = io.open("./WhWz & RR Translation - RR_ Server-Side Text.tsv")
-- assert(infd, "Please download the spreadsheet as 'WhWz & RR Translation - RR_ Server-Side Text.tsv'")
-- local body = infd:read("*a")
-- infd:close()

local split = utils.split_by_pattern(body, "\n")

-- Remove percentages, we don't care about them.
table.remove(split, 1)

local langs = {}

-- List of languages
local langs_split = utils.split_by_pattern(table.remove(split, 1), "\t")
-- Remove 'Fields' marker at start
table.remove(langs_split, 1)
-- Remove 'Error Code' marker at start
table.remove(langs_split, 1)
for i, lang in ipairs(langs_split) do
    if lang ~= "" then
        langs[i] = lang
    end
end

local messages = {}

for _, line in ipairs(split) do
    local translations = utils.split_by_pattern(line, "\t")
    local message = table.remove(translations, 1)
    local error_code = table.remove(translations, 1)

    messages[message] = {
        error_code = error_code,
        langs = {},
    }

    for i, translation in ipairs(translations) do
        if langs[i] then
            messages[message].langs[langs[i]] = translation
        end
    end
end

local output_lines = {
    "package gpcm",
    "",
    "var (",
}

for _, message_name in ipairs(ORDERED_MESSAGES) do
    local message = messages[message_name]
    assert(message, "Missing message for " .. message_name)
    table.insert(output_lines, string.format("\t%s = WWFCErrorMessage{", message_name))
    table.insert(output_lines, string.format("\t\tErrorCode: %s,", message.error_code))
    table.insert(output_lines, "\t\tMessageRMC: map[byte]string{")

    for _, lang in ipairs(ORDERED_LANGUAGES) do
        local mapped = SHEET_LANG_TO_WWFC_LANG[lang]
        local translation = message.langs[lang]
        if mapped and translation ~= "" then
            table.insert(output_lines, string.format('\t\t\t%s: "" +', mapped))

            local splits = utils.split_by_pattern(translation, "+")
            for i, v in ipairs(splits) do
                local segment = utils.trim(v)
                table.insert(output_lines, string.format("\t\t\t\t%s%s", segment, i == #splits and "" or " +"))
            end
        end
    end

    table.insert(output_lines, "\t\t},")
    table.insert(output_lines, "\t}")
    table.insert(output_lines, "")
end

table.insert(output_lines, ")")
local output = table.concat(output_lines, "\n")

local fd = io.open("../gpcm/error_messages.go", "w+")
assert(fd, "Unable to open error_messages.go")

fd:write(output)
fd:close()
