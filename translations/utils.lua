local M = {}

--- @param str string
--- @param pattern string
--- @return string[]
function M.split_by_pattern(str, pattern)
    local segments = {}

    local old_pos = 1
    local pos, end_pos, capt = str:find(pattern)

    if not pos then
        return { str }
    end

    while pos do
        if capt then
            table.insert(segments, capt)
        else
            table.insert(segments, str:sub(old_pos, pos - 1))
        end

        old_pos = end_pos + 1
        pos, end_pos, capt = str:find(pattern, end_pos + 1)
    end

    if old_pos < #str then
        table.insert(segments, str:sub(old_pos))
    end

    return segments
end

--- @param str string
--- @return string
function M.trim(str) return str:match("^()%s*$") and "" or str:match("^%s*(.*%S)") end

return M
