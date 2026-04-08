local Layout = require("nui.layout")
local Popup = require("nui.popup")
local api = require("mairu.api")

local M = {}

local chat_layout = nil
local top_popup = nil
local bottom_popup = nil

local function create_chat_ui()
  top_popup = Popup({
    border = {
      style = "rounded",
      text = {
        top = " Mairu Chat History ",
        top_align = "center",
      },
    },
    buf_options = {
      modifiable = true,
      readonly = true,
      filetype = "markdown",
    },
  })

  bottom_popup = Popup({
    enter = true,
    border = {
      style = "rounded",
      text = {
        top = " Input (<C-s> to Send, q to Close) ",
        top_align = "center",
      },
    },
    buf_options = {
      modifiable = true,
      readonly = false,
      filetype = "markdown",
    },
  })

  chat_layout = Layout(
    {
      position = "50%",
      size = {
        width = "80%",
        height = "80%",
      },
    },
    Layout.Box({
      Layout.Box(top_popup, { size = "80%" }),
      Layout.Box(bottom_popup, { size = "20%" }),
    }, { dir = "col" })
  )
end

local function append_to_history(text, is_user)
  if not top_popup or not top_popup.bufnr then return end
  
  -- temporarily make modifiable
  vim.api.nvim_set_option_value("modifiable", true, { buf = top_popup.bufnr })
  vim.api.nvim_set_option_value("readonly", false, { buf = top_popup.bufnr })
  
  local lines = vim.split(text, "\n")
  local prefix = is_user and "**User:**" or "**Mairu:**"
  
  table.insert(lines, 1, "")
  table.insert(lines, 2, prefix)
  
  -- Append to buffer
  local line_count = vim.api.nvim_buf_line_count(top_popup.bufnr)
  vim.api.nvim_buf_set_lines(top_popup.bufnr, line_count, line_count, false, lines)
  
  -- Scroll to bottom
  vim.api.nvim_win_set_cursor(top_popup.winid, { vim.api.nvim_buf_line_count(top_popup.bufnr), 0 })
  
  -- restore readonly
  vim.api.nvim_set_option_value("modifiable", false, { buf = top_popup.bufnr })
  vim.api.nvim_set_option_value("readonly", true, { buf = top_popup.bufnr })
end

local function append_raw(text)
  if not top_popup or not top_popup.bufnr then return end
  
  vim.api.nvim_set_option_value("modifiable", true, { buf = top_popup.bufnr })
  vim.api.nvim_set_option_value("readonly", false, { buf = top_popup.bufnr })
  
  local lines = vim.split(text, "\n")
  local line_count = vim.api.nvim_buf_line_count(top_popup.bufnr)
  vim.api.nvim_buf_set_lines(top_popup.bufnr, line_count, line_count, false, lines)
  vim.api.nvim_win_set_cursor(top_popup.winid, { vim.api.nvim_buf_line_count(top_popup.bufnr), 0 })
  
  vim.api.nvim_set_option_value("modifiable", false, { buf = top_popup.bufnr })
  vim.api.nvim_set_option_value("readonly", true, { buf = top_popup.bufnr })
end

local function send_query()
  if not bottom_popup or not bottom_popup.bufnr then return end
  
  local lines = vim.api.nvim_buf_get_lines(bottom_popup.bufnr, 0, -1, false)
  local query = table.concat(lines, "\n"):gsub("^%s*(.-)%s*$", "%1")
  
  if query == "" then return end
  
  -- Clear input
  vim.api.nvim_buf_set_lines(bottom_popup.bufnr, 0, -1, false, {})
  
  append_to_history(query, true)
  append_raw("Thinking...")
  
  api.vibe_query(query, function(data, err)
    if not top_popup or not top_popup.bufnr then return end
    
    -- Remove the "Thinking..." line
    vim.api.nvim_set_option_value("modifiable", true, { buf = top_popup.bufnr })
    vim.api.nvim_set_option_value("readonly", false, { buf = top_popup.bufnr })
    
    local line_count = vim.api.nvim_buf_line_count(top_popup.bufnr)
    vim.api.nvim_buf_set_lines(top_popup.bufnr, line_count - 1, line_count, false, {})
    
    vim.api.nvim_set_option_value("modifiable", false, { buf = top_popup.bufnr })
    vim.api.nvim_set_option_value("readonly", true, { buf = top_popup.bufnr })
    
    if err then
      append_raw("Error: " .. err)
      return
    end
    
    -- vibe_query returns { reasoning = "...", results = [ { store, items } ] }
    local reply = ""
    if data.reasoning then
      reply = reply .. data.reasoning .. "\n\n"
    end
    
    local found_any = false
    if data.results then
      for _, group in ipairs(data.results) do
        if group.items and #group.items > 0 then
          found_any = true
          reply = reply .. "**" .. group.store:gsub("^%l", string.upper) .. " Matches:**\n"
          for _, item in ipairs(group.items) do
             local label = item.content or item.name or item.abstract or item.uri or item.id or "Unknown"
             -- Truncate label if it's too long
             if string.len(label) > 100 then
               label = string.sub(label, 1, 100) .. "..."
             end
             -- Format on one line
             reply = reply .. "- " .. label:gsub("\n", " ") .. "\n"
          end
          reply = reply .. "\n"
        end
      end
    end
    
    if not found_any then
      reply = reply .. "*(No matching context found)*"
    end
    
    append_to_history(reply, false)
    

  end)
end

function M.open()
  if chat_layout then
    chat_layout:show()
    return
  end
  
  create_chat_ui()
  chat_layout:mount()
  
  -- Initial text
  vim.api.nvim_set_option_value("modifiable", true, { buf = top_popup.bufnr })
  vim.api.nvim_set_option_value("readonly", false, { buf = top_popup.bufnr })
  
  vim.api.nvim_buf_set_lines(top_popup.bufnr, 0, -1, false, { "Welcome to Mairu Chat." })
  
  vim.api.nvim_set_option_value("modifiable", false, { buf = top_popup.bufnr })
  vim.api.nvim_set_option_value("readonly", true, { buf = top_popup.bufnr })
  
  -- Mappings
  bottom_popup:map("i", "<C-s>", function()
    send_query()
    -- stay in insert mode
  end, { noremap = true })
  
  bottom_popup:map("n", "<C-s>", function()
    send_query()
  end, { noremap = true })
  
  local close_fn = function()
    chat_layout:unmount()
    chat_layout = nil
  end
  
  bottom_popup:map("n", "q", close_fn, { noremap = true })
  top_popup:map("n", "q", close_fn, { noremap = true })
end

return M
