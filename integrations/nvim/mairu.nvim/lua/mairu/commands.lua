local api = require("mairu.api")

local M = {}

local function store_memory()
  vim.ui.input({ prompt = "Memory to store: " }, function(input)
    if not input or input == "" then return end
    
    api.store_memory(input, function(data, err)
      if err then
        vim.notify("Failed to store memory: " .. err, vim.log.levels.ERROR, { title = "Mairu" })
      else
        vim.notify("Memory stored successfully.", vim.log.levels.INFO, { title = "Mairu" })
      end
    end)
  end)
end

local function search_context()
  vim.ui.input({ prompt = "Search Query: " }, function(input)
    if not input or input == "" then return end
    
    require("mairu.ui.popup").open_search(input)
  end)
end

local function vibe_mutation()
  vim.ui.input({ prompt = "Mutation Plan: " }, function(input)
    if not input or input == "" then return end
    
    api.vibe_mutation(input, function(data, err)
      if err then
        vim.notify("Failed to execute mutation: " .. err, vim.log.levels.ERROR, { title = "Mairu" })
      else
        vim.notify("Mutation applied successfully.", vim.log.levels.INFO, { title = "Mairu" })
      end
    end)
  end)
end

local function find_symbol_impact()
  vim.ui.input({ prompt = "Symbol or Node to find blast radius for: " }, function(input)
    if not input or input == "" then return end
    require("mairu.ui.popup").open_search(input) -- For now, standard search
  end)
end

function M.command_palette()
  local commands = {
    { display = "Search Context (Contextual Search)", cmd = search_context },
    { display = "Find Symbol Impact (Blast Radius)", cmd = find_symbol_impact },
    { display = "Store Memory (Natural Language)", cmd = store_memory },
    { display = "Mairu Update Knowledge (Vibe Mutation)", cmd = vibe_mutation },
    { display = "Toggle Ambient Context Sidebar", cmd = require("mairu.ui.sidebar").toggle },
    { display = "Open Mairu Chat", cmd = require("mairu.ui.chat").open },
  }
  
  vim.ui.select(commands, {
    prompt = "Mairu Commands",
    format_item = function(item)
      return item.display
    end,
  }, function(choice)
    if choice then
      choice.cmd()
    end
  end)
end

return M