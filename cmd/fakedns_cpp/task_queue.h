#pragma once

#include <condition_variable>
#include <cstddef>
#include <deque>
#include <mutex>
#include <utility>

namespace fakedns {

template <typename T>
class TaskQueue final {
 public:
  explicit TaskQueue(std::size_t max_size) : max_size_(max_size) {}

  bool Push(T&& task) {
    std::unique_lock<std::mutex> lock(mutex_);
    cv_not_full_.wait(lock, [this] { return stopped_ || queue_.size() < max_size_; });
    if (stopped_) {
      return false;
    }
    queue_.push_back(std::move(task));
    cv_not_empty_.notify_one();
    return true;
  }

  bool Pop(T* out) {
    std::unique_lock<std::mutex> lock(mutex_);
    cv_not_empty_.wait(lock, [this] { return stopped_ || !queue_.empty(); });
    if (queue_.empty()) {
      return false;
    }
    *out = std::move(queue_.front());
    queue_.pop_front();
    cv_not_full_.notify_one();
    return true;
  }

  void Stop() {
    {
      std::lock_guard<std::mutex> lock(mutex_);
      stopped_ = true;
    }
    cv_not_empty_.notify_all();
    cv_not_full_.notify_all();
  }

 private:
  std::mutex mutex_;
  std::condition_variable cv_not_empty_;
  std::condition_variable cv_not_full_;
  std::deque<T> queue_;
  std::size_t max_size_ = 0;
  bool stopped_ = false;
};

}  // namespace fakedns
