local config = require("mairu.config")

local M = {}
local job_id = nil

function M.start()
  if job_id then
    vim.notify("Mairu context server is already running.", vim.log.levels.INFO, { title = "Mairu" })
    return
  end

  local bin = config.options.server.bin_path
  
  -- Expand path if needed (e.g. if someone uses ~)
  bin = vim.fn.expand(bin)
  
  -- Simple check if executable is available
  if vim.fn.executable(bin) == 0 then
    vim.notify("Mairu agent executable not found: " .. bin .. "\nPlease ensure it's built and in PATH.", vim.log.levels.ERROR, { title = "Mairu" })
    return
  end

  local port = config.options.server.port
  local cmd = { bin, "context-server", "-p", tostring(port) }

  job_id = vim.fn.jobstart(cmd, {
    on_stdout = function(_, data, _)
      -- Log to a file or buffer if debugging is needed
    end,
    on_stderr = function(_, data, _)
      -- Log errors
    end,
    on_exit = function(_, code, _)
      job_id = nil
      if code ~= 0 and code ~= 143 then -- 143 is SIGTERM
        vim.notify("Mairu context server exited with code " .. tostring(code), vim.log.levels.WARN, { title = "Mairu" })
      end
    end,
  })

  if job_id <= 0 then
    vim.notify("Failed to start Mairu context server.", vim.log.levels.ERROR, { title = "Mairu" })
    job_id = nil
  else
    vim.notify("Mairu context server started on port " .. tostring(port), vim.log.levels.INFO, { title = "Mairu" })
  end
end

function M.stop()
  if job_id then
    vim.fn.jobstop(job_id)
    job_id = nil
    vim.notify("Mairu context server stopped.", vim.log.levels.INFO, { title = "Mairu" })
  end
end

function M.is_running()
  return job_id ~= nil
end

return M
